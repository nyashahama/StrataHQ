// backend/internal/earlyaccess/service.go
package earlyaccess

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

var ErrNotFound = errors.New("early access request not found")

type SubmitParams struct {
	FullName   string
	Email      string
	SchemeName string
	UnitCount  int32
}

type RequestResponse struct {
	ReviewedAt *string `json:"reviewed_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	ID         string  `json:"id"`
	FullName   string  `json:"full_name"`
	Email      string  `json:"email"`
	SchemeName string  `json:"scheme_name"`
	Status     string  `json:"status"`
	UnitCount  int32   `json:"unit_count"`
}

type Servicer interface {
	Submit(ctx context.Context, p SubmitParams) (*RequestResponse, error)
	List(ctx context.Context) ([]RequestResponse, error)
	Approve(ctx context.Context, id string) (*RequestResponse, error)
	Reject(ctx context.Context, id string) (*RequestResponse, error)
}

type Service struct {
	db          *dbgen.Queries
	authService auth.Servicer
	notifier    notification.Sender
	appBaseURL  string
}

func NewService(db *dbgen.Queries, authService auth.Servicer, notifier notification.Sender, appBaseURL string) *Service {
	return &Service{db: db, authService: authService, notifier: notifier, appBaseURL: appBaseURL}
}

func (s *Service) Submit(ctx context.Context, p SubmitParams) (*RequestResponse, error) {
	row, err := s.db.CreateEarlyAccessRequest(ctx, dbgen.CreateEarlyAccessRequestParams{
		FullName:   p.FullName,
		Email:      p.Email,
		SchemeName: p.SchemeName,
		UnitCount:  p.UnitCount,
	})
	if err != nil {
		return nil, err
	}
	return toResponse(row), nil
}

func (s *Service) List(ctx context.Context) ([]RequestResponse, error) {
	rows, err := s.db.ListEarlyAccessRequests(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RequestResponse, len(rows))
	for i, r := range rows {
		out[i] = *toResponse(r)
	}
	return out, nil
}

func (s *Service) Approve(ctx context.Context, id string) (*RequestResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}

	req, err := s.db.GetEarlyAccessRequest(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Create account with random temp password (user will reset it)
	tempPass := make([]byte, 16)
	if _, err := rand.Read(tempPass); err != nil {
		return nil, err
	}
	_, regErr := s.authService.Register(ctx, req.Email, hex.EncodeToString(tempPass), req.FullName)
	if regErr != nil && !errors.Is(regErr, auth.ErrEmailExists) {
		return nil, regErr
	}

	// Generate password reset URL without sending reset email
	setPasswordURL, err := s.authService.IssuePasswordResetURL(ctx, req.Email, s.appBaseURL)
	if err != nil {
		return nil, err
	}

	// Send approval email with the set-password link
	_ = s.notifier.SendEarlyAccessApproval(ctx, req.Email, req.FullName, setPasswordURL)

	// Mark approved
	updated, err := s.db.UpdateEarlyAccessStatus(ctx, dbgen.UpdateEarlyAccessStatusParams{
		ID:     uid,
		Status: dbgen.EarlyAccessStatusApproved,
	})
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func (s *Service) Reject(ctx context.Context, id string) (*RequestResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	updated, err := s.db.UpdateEarlyAccessStatus(ctx, dbgen.UpdateEarlyAccessStatusParams{
		ID:     uid,
		Status: dbgen.EarlyAccessStatusRejected,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toResponse(updated), nil
}

func toResponse(r dbgen.EarlyAccessRequest) *RequestResponse {
	resp := &RequestResponse{
		ID:         r.ID.String(),
		FullName:   r.FullName,
		Email:      r.Email,
		SchemeName: r.SchemeName,
		UnitCount:  r.UnitCount,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.ReviewedAt.Valid {
		s := r.ReviewedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		resp.ReviewedAt = &s
	}
	return resp
}
