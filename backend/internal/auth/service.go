package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/platform/database"
)

// Sentinel errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// Response types returned by the service.

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type OrgInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

type MeResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Org      OrgInfo `json:"org"`
	Role     string  `json:"role"`
}

// Servicer is the interface Handler depends on (enables test mocks).
type Servicer interface {
	Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, cache *redis.Client, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *Service) Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error) {
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
		org, txErr = q.CreateOrg(ctx, orgName)
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

	return &MeResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Org:      OrgInfo{ID: org.ID.String(), Name: org.Name},
		Role:     membership.Role,
	}, nil
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
		},
	}, nil
}
