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
	"github.com/stratahq/backend/internal/audit"
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

type queryStore interface {
	CreateInvitation(ctx context.Context, arg dbgen.CreateInvitationParams) (dbgen.Invitation, error)
	ListInvitationsByOrg(ctx context.Context, orgID uuid.UUID) ([]dbgen.Invitation, error)
	GetInvitationByID(ctx context.Context, id uuid.UUID) (dbgen.Invitation, error)
	UpdateInvitationToken(ctx context.Context, arg dbgen.UpdateInvitationTokenParams) (dbgen.Invitation, error)
	UpdateInvitationStatus(ctx context.Context, arg dbgen.UpdateInvitationStatusParams) error
	GetInvitationByToken(ctx context.Context, token string) (dbgen.Invitation, error)
	GetUserByEmail(ctx context.Context, email string) (dbgen.User, error)
	CreateRefreshToken(ctx context.Context, arg dbgen.CreateRefreshTokenParams) (dbgen.RefreshToken, error)
	GetScheme(ctx context.Context, id uuid.UUID) (dbgen.Scheme, error)
	GetUnit(ctx context.Context, id uuid.UUID) (dbgen.Unit, error)
}

type txStore interface {
	CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error)
	CreateOrgMembership(ctx context.Context, arg dbgen.CreateOrgMembershipParams) (dbgen.OrgMembership, error)
	UpsertSchemeMembership(ctx context.Context, arg dbgen.UpsertSchemeMembershipParams) (dbgen.SchemeMembership, error)
	UpdateInvitationStatus(ctx context.Context, arg dbgen.UpdateInvitationStatusParams) error
}

type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	q             queryStore
	withTx        func(ctx context.Context, fn func(q txStore) error) error
	db            *database.Pool
	sender        notification.Sender
	appBaseURL    string
	jwtSecret     string
	jwtIssuer     string
	jwtAudience   string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
	auditor       resourceAuditor
}

func NewService(db *database.Pool, sender notification.Sender, appBaseURL, jwtSecret, jwtIssuer, jwtAudience string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return NewServiceWithAudit(db, sender, appBaseURL, jwtSecret, jwtIssuer, jwtAudience, jwtExpiry, refreshExpiry, nil)
}

func NewServiceWithAudit(db *database.Pool, sender notification.Sender, appBaseURL, jwtSecret, jwtIssuer, jwtAudience string, jwtExpiry, refreshExpiry time.Duration, auditor resourceAuditor) *Service {
	issuer, audience := auth.ResolveJWTConfig(appBaseURL, jwtIssuer, jwtAudience)
	return &Service{
		q: db.Q,
		withTx: func(ctx context.Context, fn func(q txStore) error) error {
			return database.WithTxQueries(ctx, db, func(q *dbgen.Queries) error {
				return fn(q)
			})
		},
		db:            db,
		sender:        sender,
		appBaseURL:    appBaseURL,
		jwtSecret:     jwtSecret,
		jwtIssuer:     issuer,
		jwtAudience:   audience,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
		auditor:       auditor,
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
	unitID, err := unitIDFromString(p.UnitID)
	if err != nil {
		return nil, err
	}
	if validationErr := s.validateInvitationScope(ctx, oid, sid, unitID); validationErr != nil {
		return nil, validationErr
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	inv, err := s.q.CreateInvitation(ctx, dbgen.CreateInvitationParams{
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

	if s.auditor != nil {
		unitIDStr := ""
		if unitID.Valid {
			unitIDStr = uuid.UUID(unitID.Bytes).String()
		}
		_ = s.auditor.RecordResourceEvent(ctx, invitationCreatedAuditEvent(invitationAuditInput{
			OrgID:        oid.String(),
			SchemeID:     sid.String(),
			ActorRole:    "admin",
			InvitationID: inv.ID.String(),
			Email:        inv.Email,
			FullName:     inv.FullName,
			Role:         inv.Role,
			UnitID:       unitIDStr,
			Status:       inv.Status,
			ExpiresAt:    inv.ExpiresAt,
		}))
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
	invs, err := s.q.ListInvitationsByOrg(ctx, oid)
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

	existing, err := s.q.GetInvitationByID(ctx, iid)
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

	inv, err := s.q.UpdateInvitationToken(ctx, dbgen.UpdateInvitationTokenParams{
		Token:     token,
		ExpiresAt: expiresAt,
		ID:        iid,
	})
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		unitIDStr := ""
		if inv.UnitID.Valid {
			unitIDStr = uuid.UUID(inv.UnitID.Bytes).String()
		}
		_ = s.auditor.RecordResourceEvent(ctx, invitationResentAuditEvent(invitationAuditInput{
			OrgID:        inv.OrgID.String(),
			SchemeID:     inv.SchemeID.String(),
			ActorRole:    "admin",
			InvitationID: inv.ID.String(),
			Email:        inv.Email,
			FullName:     inv.FullName,
			Role:         inv.Role,
			UnitID:       unitIDStr,
			Status:       inv.Status,
			ExpiresAt:    inv.ExpiresAt,
		}))
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

	existing, err := s.q.GetInvitationByID(ctx, iid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if existing.OrgID != oid {
		return ErrForbidden
	}

	if err := s.q.UpdateInvitationStatus(ctx, dbgen.UpdateInvitationStatusParams{
		Status: "revoked",
		ID:     iid,
	}); err != nil {
		return err
	}

	if s.auditor != nil {
		unitIDStr := ""
		if existing.UnitID.Valid {
			unitIDStr = uuid.UUID(existing.UnitID.Bytes).String()
		}
		_ = s.auditor.RecordResourceEvent(ctx, invitationRevokedAuditEvent(invitationAuditInput{
			OrgID:        existing.OrgID.String(),
			SchemeID:     existing.SchemeID.String(),
			ActorRole:    "admin",
			InvitationID: existing.ID.String(),
			Email:        existing.Email,
			FullName:     existing.FullName,
			Role:         existing.Role,
			UnitID:       unitIDStr,
			Status:       "revoked",
			ExpiresAt:    existing.ExpiresAt,
		}))
	}

	return nil
}

func (s *Service) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	inv, err := s.q.GetInvitationByToken(ctx, token)
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
	inv, err := s.q.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if inv.Status != "pending" || inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	_, err = s.q.GetUserByEmail(ctx, inv.Email)
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

	err = s.withTx(ctx, func(q txStore) error {
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

	accessToken, err := auth.GenerateAccessToken(user.ID.String(), inv.OrgID.String(), inv.Role, s.jwtIssuer, s.jwtAudience, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	err = s.storeRefreshToken(ctx, user.ID, refreshToken)
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

func (s *Service) validateInvitationScope(ctx context.Context, orgID, schemeID uuid.UUID, unitID pgtype.UUID) error {
	scheme, err := s.q.GetScheme(ctx, schemeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("invalid scheme_id")
		}
		return err
	}
	if scheme.OrgID != orgID {
		return ErrForbidden
	}
	if !unitID.Valid {
		return nil
	}

	unit, err := s.q.GetUnit(ctx, unitID.Bytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("invalid unit_id")
		}
		return err
	}
	if unit.SchemeID != schemeID {
		return errors.New("invalid unit_id")
	}
	return nil
}

func (s *Service) storeRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	_, err := s.q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		Token:     auth.HashRefreshToken(refreshToken),
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	})
	return err
}

func unitIDFromString(raw string) (pgtype.UUID, error) {
	if raw == "" {
		return pgtype.UUID{}, nil
	}
	uid, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}, errors.New("invalid unit_id")
	}
	return pgtype.UUID{Bytes: uid, Valid: true}, nil
}

type invitationAuditInput struct {
	OrgID        string
	SchemeID     string
	ActorUserID  string
	ActorRole    string
	InvitationID string
	Email        string
	FullName     string
	Role         string
	UnitID       string
	Status       string
	ExpiresAt    time.Time
}

func invitationCreatedAuditEvent(input invitationAuditInput) audit.ResourceEvent {
	afterState := map[string]any{
		"email":      input.Email,
		"full_name":  input.FullName,
		"role":       input.Role,
		"scheme_id":  input.SchemeID,
		"status":     input.Status,
		"expires_at": input.ExpiresAt.Format(time.RFC3339),
	}
	if input.UnitID != "" {
		afterState["unit_id"] = input.UnitID
	}
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "invitation",
		ResourceID:   input.InvitationID,
		Action:       "invitation.created",
		AfterState:   afterState,
	}
}

func invitationResentAuditEvent(input invitationAuditInput) audit.ResourceEvent {
	afterState := map[string]any{
		"email":      input.Email,
		"full_name":  input.FullName,
		"role":       input.Role,
		"scheme_id":  input.SchemeID,
		"status":     input.Status,
		"expires_at": input.ExpiresAt.Format(time.RFC3339),
	}
	if input.UnitID != "" {
		afterState["unit_id"] = input.UnitID
	}
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "invitation",
		ResourceID:   input.InvitationID,
		Action:       "invitation.resent",
		AfterState:   afterState,
	}
}

func invitationRevokedAuditEvent(input invitationAuditInput) audit.ResourceEvent {
	afterState := map[string]any{
		"email":      input.Email,
		"full_name":  input.FullName,
		"role":       input.Role,
		"scheme_id":  input.SchemeID,
		"status":     input.Status,
		"expires_at": input.ExpiresAt.Format(time.RFC3339),
	}
	if input.UnitID != "" {
		afterState["unit_id"] = input.UnitID
	}
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "invitation",
		ResourceID:   input.InvitationID,
		Action:       "invitation.revoked",
		AfterState:   afterState,
	}
}
