package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
)

// Sentinel errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrWrongPassword      = errors.New("current password is incorrect")
)

// Response types returned by the service.

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone,omitempty"`
}

type OrgInfo struct {
	ContactEmail *string `json:"contact_email"`
	ContactPhone *string `json:"contact_phone"`
	ID           string  `json:"id"`
	Name         string  `json:"name"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
	ExpiresIn    int      `json:"expires_in"` // seconds
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type SchemeMembership struct {
	SchemeID       string  `json:"scheme_id"`
	SchemeName     string  `json:"scheme_name"`
	UnitID         *string `json:"unit_id"`
	UnitIdentifier *string `json:"unit_identifier"`
	Role           string  `json:"role"`
}

//nolint:govet // Keep the response DTO grouped by API meaning rather than field packing.
type MeResponse struct {
	SchemeMemberships []SchemeMembership `json:"scheme_memberships"`
	Org               OrgInfo            `json:"org"`
	ID                string             `json:"id"`
	Email             string             `json:"email"`
	FullName          string             `json:"full_name"`
	Phone             *string            `json:"phone"`
	Role              string             `json:"role"`
	WizardComplete    bool               `json:"wizard_complete"`
}

type SetupResponse struct {
	Org    OrgInfo `json:"org"`
	Scheme struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scheme"`
}

// Servicer is the interface Handler depends on (enables test mocks).
type Servicer interface {
	Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
	Setup(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, password string) error
	IssuePasswordResetURL(ctx context.Context, email, appBaseURL string) (string, error)
	UpdateProfile(ctx context.Context, userID, orgID, email, fullName string, phone *string) (*MeResponse, error)
	UpdateOrg(ctx context.Context, orgID, name string, contactEmail, contactPhone *string) (*OrgInfo, error)
	ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	sender        notification.Sender
	appBaseURL    string
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, cache *redis.Client, sender notification.Sender, jwtSecret, appBaseURL string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		sender:        sender,
		jwtSecret:     jwtSecret,
		appBaseURL:    appBaseURL,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *Service) Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	_, err := s.db.Q.GetUserByEmail(ctx, email)
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
	var org dbgen.Org

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			FullName:     fullName,
		})
		if txErr != nil {
			return txErr
		}
		org, txErr = q.CreateOrg(ctx, "") // name set during onboarding
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  org.ID,
			Role:   "admin",
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, org.ID.String(), "admin")
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, ErrInvalidCredentials
	}

	m := memberships[0]
	return s.issueTokens(ctx, user, m.OrgID.String(), m.Role)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	rt, err := s.db.Q.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, ErrInvalidToken
	}
	m := memberships[0]

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.RevokeRefreshToken(ctx, refreshToken); txErr != nil {
			return txErr
		}
		_, txErr := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
			Token:     newRefreshToken,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(s.refreshExpiry),
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := GenerateAccessToken(user.ID.String(), m.OrgID.String(), m.Role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.db.Q.RevokeRefreshToken(ctx, refreshToken)
}

func (s *Service) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	org, err := s.db.Q.GetOrg(ctx, oid)
	if err != nil {
		return nil, err
	}
	membership, err := s.db.Q.GetOrgMembershipByUser(ctx, dbgen.GetOrgMembershipByUserParams{
		UserID: uid,
		OrgID:  oid,
	})
	if err != nil {
		return nil, err
	}

	resp := &MeResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Phone:    textPointer(user.Phone),
		Org: OrgInfo{
			ID:           org.ID.String(),
			Name:         org.Name,
			ContactEmail: textPointer(org.ContactEmail),
			ContactPhone: textPointer(org.ContactPhone),
		},
		Role: membership.Role,
	}

	if membership.Role == "admin" {
		schemes, err := s.db.Q.ListSchemesByOrg(ctx, oid)
		if err != nil {
			return nil, err
		}
		resp.WizardComplete = len(schemes) > 0
		for _, sc := range schemes {
			resp.SchemeMemberships = append(resp.SchemeMemberships, SchemeMembership{
				SchemeID:       sc.ID.String(),
				SchemeName:     sc.Name,
				UnitID:         nil,
				UnitIdentifier: nil,
				Role:           "admin",
			})
		}
	} else {
		resp.WizardComplete = true
		memberships, err := s.db.Q.ListSchemeMembershipsByUser(ctx, uid)
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			sm := SchemeMembership{
				SchemeID:   m.SchemeID.String(),
				SchemeName: m.SchemeName,
				Role:       m.Role,
			}
			if m.UnitID.Valid {
				unitStr := uuid.UUID(m.UnitID.Bytes).String()
				sm.UnitID = &unitStr
			}
			if m.UnitIdentifier.Valid {
				identifier := m.UnitIdentifier.String
				sm.UnitIdentifier = &identifier
			}
			resp.SchemeMemberships = append(resp.SchemeMemberships, sm)
		}
	}

	if resp.SchemeMemberships == nil {
		resp.SchemeMemberships = []SchemeMembership{}
	}

	return resp, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID, orgID, email, fullName string, phone *string) (*MeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.UpdateUserProfile(ctx, dbgen.UpdateUserProfileParams{
		ID:       uid,
		Email:    email,
		FullName: fullName,
		Phone:    textValue(phone),
	})
	if err != nil {
		if isUniqueEmailViolation(err) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	return s.Me(ctx, user.ID.String(), orgID)
}

func (s *Service) UpdateOrg(ctx context.Context, orgID, name string, contactEmail, contactPhone *string) (*OrgInfo, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	org, err := s.db.Q.UpdateOrg(ctx, dbgen.UpdateOrgParams{
		ID:           oid,
		Name:         name,
		ContactEmail: textValue(contactEmail),
		ContactPhone: textValue(contactPhone),
	})
	if err != nil {
		return nil, err
	}

	return &OrgInfo{
		ID:           org.ID.String(),
		Name:         org.Name,
		ContactEmail: textPointer(org.ContactEmail),
		ContactPhone: textPointer(org.ContactPhone),
	}, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, uid)
	if err != nil {
		return err
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); compareErr != nil {
		return ErrWrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(nextPassword), 12)
	if err != nil {
		return err
	}

	return database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID:           uid,
			PasswordHash: string(hash),
		}); txErr != nil {
			return txErr
		}
		return q.RevokeAllUserRefreshTokens(ctx, uid)
	})
}

func (s *Service) Setup(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org id: %w", err)
	}

	var resp SetupResponse

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		orgRow, txErr := q.UpdateOrg(ctx, dbgen.UpdateOrgParams{
			Name:         orgName,
			ContactEmail: pgtype.Text{String: contactEmail, Valid: true},
			ContactPhone: pgtype.Text{},
			ID:           oid,
		})
		if txErr != nil {
			return txErr
		}
		resp.Org = OrgInfo{
			ID:           orgRow.ID.String(),
			Name:         orgRow.Name,
			ContactEmail: textPointer(orgRow.ContactEmail),
			ContactPhone: textPointer(orgRow.ContactPhone),
		}

		scheme, txErr := q.CreateScheme(ctx, dbgen.CreateSchemeParams{
			OrgID:     oid,
			Name:      schemeName,
			Address:   schemeAddress,
			UnitCount: unitCount,
		})
		if txErr != nil {
			return txErr
		}
		resp.Scheme.ID = scheme.ID.String()
		resp.Scheme.Name = scheme.Name
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // silent — no enumeration
	}
	if err != nil {
		return err
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)

	key := "pwreset:" + token
	if err := s.cache.Set(ctx, key, user.ID.String(), time.Hour).Err(); err != nil {
		return err
	}

	resetURL := s.appBaseURL + "/auth/reset-password?token=" + token
	return s.sender.SendPasswordReset(ctx, user.Email, resetURL)
}

// IssuePasswordResetURL generates a password reset token and returns the URL
// without sending any email. Used by earlyaccess approval flow.
func (s *Service) IssuePasswordResetURL(ctx context.Context, email, appBaseURL string) (string, error) {
	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	key := "pwreset:" + token
	if err := s.cache.Set(ctx, key, user.ID.String(), 24*time.Hour).Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/auth/reset-password?token=%s", appBaseURL, token), nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	key := "pwreset:" + token
	userIDStr, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		return ErrInvalidToken
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID:           uid,
			PasswordHash: string(hash),
		}); txErr != nil {
			return txErr
		}
		return q.RevokeAllUserRefreshTokens(ctx, uid)
	})
	if err != nil {
		return err
	}

	s.cache.Del(ctx, key)
	return nil
}

// issueTokens creates a token pair and persists the refresh token.
func (s *Service) issueTokens(ctx context.Context, user dbgen.User, orgID, role string) (*AuthResponse, error) {
	accessToken, err := GenerateAccessToken(user.ID.String(), orgID, role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken()
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

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
		User: UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
			Phone:    textString(user.Phone),
		},
	}, nil
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func isUniqueEmailViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
