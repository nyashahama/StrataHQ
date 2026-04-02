// backend/internal/invitation/service.go
package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrNotFound     = errors.New("invitation not found")
	ErrForbidden    = errors.New("invitation belongs to a different org")
	ErrInvalidToken = errors.New("invalid, expired, or already used invitation token")
	ErrEmailExists  = errors.New("email already registered")
)

type CreateParams struct {
	Email    string
	FullName string
	Role     string // trustee | resident
	SchemeID string
	UnitID   string // required for resident, empty for trustee
}

type InvitationResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	SchemeID  string    `json:"scheme_id"`
	Status    string    `json:"status"`
}

type VerifyResponse struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	SchemeID string `json:"scheme_id"`
}

type Servicer interface {
	Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error)
	List(ctx context.Context, orgID string) ([]InvitationResponse, error)
	Resend(ctx context.Context, orgID, invitationID, appBaseURL string) (*InvitationResponse, error)
	Revoke(ctx context.Context, orgID, invitationID string) error
	Verify(ctx context.Context, token string) (*VerifyResponse, error)
	Accept(ctx context.Context, token, password string) (*auth.AuthResponse, error)
}

type Service struct {
	db            *database.Pool
	sender        notification.Sender
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, sender notification.Sender, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		sender:        sender,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	sid, err := uuid.Parse(p.SchemeID)
	if err != nil {
		return nil, errors.New("invalid scheme_id")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	var unitID pgtype.UUID
	if p.UnitID != "" {
		uid, parseErr := uuid.Parse(p.UnitID)
		if parseErr != nil {
			return nil, errors.New("invalid unit_id")
		}
		unitID = pgtype.UUID{Bytes: uid, Valid: true}
	}

	inv, err := s.db.Q.CreateInvitation(ctx, dbgen.CreateInvitationParams{
		OrgID:     oid,
		SchemeID:  sid,
		UnitID:    unitID,
		Email:     p.Email,
		FullName:  p.FullName,
		Role:      p.Role,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}

	inviteURL := appBaseURL + "/auth/invite/" + token
	if err := s.sender.SendInvitation(ctx, p.Email, p.FullName, inviteURL); err != nil {
		return nil, err
	}

	return toResponse(inv), nil
}

func (s *Service) List(ctx context.Context, orgID string) ([]InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	invs, err := s.db.Q.ListInvitationsByOrg(ctx, oid)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationResponse, len(invs))
	for i, inv := range invs {
		out[i] = *toResponse(inv)
	}
	return out, nil
}

func (s *Service) Resend(ctx context.Context, orgID, invitationID, appBaseURL string) (*InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	iid, err := uuid.Parse(invitationID)
	if err != nil {
		return nil, ErrNotFound
	}

	existing, err := s.db.Q.GetInvitationByID(ctx, iid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if existing.OrgID != oid {
		return nil, ErrForbidden
	}
	if existing.Status != "pending" {
		return nil, ErrInvalidToken
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	inv, err := s.db.Q.UpdateInvitationToken(ctx, dbgen.UpdateInvitationTokenParams{
		Token:     token,
		ExpiresAt: expiresAt,
		ID:        iid,
	})
	if err != nil {
		return nil, err
	}

	inviteURL := appBaseURL + "/auth/invite/" + token
	if err := s.sender.SendInvitation(ctx, inv.Email, inv.FullName, inviteURL); err != nil {
		return nil, err
	}

	return toResponse(inv), nil
}

func (s *Service) Revoke(ctx context.Context, orgID, invitationID string) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return ErrForbidden
	}
	iid, err := uuid.Parse(invitationID)
	if err != nil {
		return ErrNotFound
	}

	existing, err := s.db.Q.GetInvitationByID(ctx, iid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if existing.OrgID != oid {
		return ErrForbidden
	}

	return s.db.Q.UpdateInvitationStatus(ctx, dbgen.UpdateInvitationStatusParams{
		Status: "revoked",
		ID:     iid,
	})
}

func (s *Service) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	inv, err := s.db.Q.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if inv.Status != "pending" || inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}
	return &VerifyResponse{
		Email:    inv.Email,
		FullName: inv.FullName,
		Role:     inv.Role,
		SchemeID: inv.SchemeID.String(),
	}, nil
}

func (s *Service) Accept(ctx context.Context, token, password string) (*auth.AuthResponse, error) {
	inv, err := s.db.Q.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if inv.Status != "pending" || inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	_, err = s.db.Q.GetUserByEmail(ctx, inv.Email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var user dbgen.User

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        inv.Email,
			PasswordHash: string(hash),
			FullName:     inv.FullName,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  inv.OrgID,
			Role:   inv.Role,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
			UserID:   user.ID,
			SchemeID: inv.SchemeID,
			UnitID:   inv.UnitID,
			Role:     inv.Role,
		})
		if txErr != nil {
			return txErr
		}
		return q.UpdateInvitationStatus(ctx, dbgen.UpdateInvitationStatusParams{
			Status: "accepted",
			ID:     inv.ID,
		})
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := auth.GenerateAccessToken(user.ID.String(), inv.OrgID.String(), inv.Role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	_, err = s.db.Q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	})
	if err != nil {
		return nil, err
	}

	return &auth.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
		User: auth.UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

func toResponse(inv dbgen.Invitation) *InvitationResponse {
	return &InvitationResponse{
		ID:        inv.ID.String(),
		Email:     inv.Email,
		FullName:  inv.FullName,
		Role:      inv.Role,
		SchemeID:  inv.SchemeID.String(),
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt,
	}
}
