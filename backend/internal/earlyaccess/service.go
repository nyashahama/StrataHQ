// backend/internal/earlyaccess/service.go
package earlyaccess

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

var (
	ErrNotFound        = errors.New("early access request not found")
	ErrForbidden       = errors.New("forbidden")
	ErrAlreadyReviewed = errors.New("early access request already reviewed")
)

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
	ApproveByToken(ctx context.Context, id, sig string, exp int64) (*RequestResponse, error)
	RejectByToken(ctx context.Context, id, sig string, exp int64) (*RequestResponse, error)
	ValidateActionToken(id, action, sig string, exp int64) error
}

type Service struct {
	db             *dbgen.Queries
	authService    auth.Servicer
	notifier       notification.Sender
	backendBaseURL string
	appBaseURL     string
	adminEmail     string
	adminSecret    string
}

func NewService(db *dbgen.Queries, authService auth.Servicer, notifier notification.Sender, backendBaseURL, appBaseURL, adminEmail, adminSecret string) *Service {
	return &Service{db: db, authService: authService, notifier: notifier, backendBaseURL: backendBaseURL, appBaseURL: appBaseURL, adminEmail: adminEmail, adminSecret: adminSecret}
}

func generateActionToken(secret, id, action string, exp int64) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(fmt.Sprintf("%s|%s|%d", id, action, exp)))
	return hex.EncodeToString(h.Sum(nil))
}

func validateActionToken(secret, id, action, sig string, exp int64) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	if time.Now().Unix() > exp {
		return false
	}
	expected := generateActionToken(secret, id, action, exp)
	return hmac.Equal([]byte(sig), []byte(expected))
}

func (s *Service) Submit(ctx context.Context, p SubmitParams) (*RequestResponse, error) {
	pending, err := s.getPendingByEmail(ctx, p.Email)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return toResponse(*pending), nil
	}

	row, err := s.db.CreateEarlyAccessRequest(ctx, dbgen.CreateEarlyAccessRequestParams{
		FullName:   p.FullName,
		Email:      p.Email,
		SchemeName: p.SchemeName,
		UnitCount:  p.UnitCount,
	})
	if err != nil {
		if isUniqueViolation(err) {
			pending, lookupErr := s.getPendingByEmail(ctx, p.Email)
			if lookupErr == nil && pending != nil {
				return toResponse(*pending), nil
			}
			if lookupErr != nil {
				return nil, lookupErr
			}
		}
		return nil, err
	}

	// Fire admin notification if configured
	if s.adminEmail != "" && s.adminSecret != "" {
		exp := time.Now().Add(7 * 24 * time.Hour).Unix()
		approveSig := generateActionToken(s.adminSecret, row.ID.String(), "approve", exp)
		rejectSig := generateActionToken(s.adminSecret, row.ID.String(), "reject", exp)
		approveURL := fmt.Sprintf("%s/api/v1/early-access/%s/approve?exp=%d&sig=%s", s.backendBaseURL, row.ID.String(), exp, approveSig)
		rejectURL := fmt.Sprintf("%s/api/v1/early-access/%s/reject?exp=%d&sig=%s", s.backendBaseURL, row.ID.String(), exp, rejectSig)
		_ = s.notifier.SendNewEarlyAccessRequest(ctx, s.adminEmail, row.FullName, row.Email, row.SchemeName, row.UnitCount, approveURL, rejectURL)
	}

	return toResponse(row), nil
}

func (s *Service) getPendingByEmail(ctx context.Context, email string) (*dbgen.EarlyAccessRequest, error) {
	rows, err := s.db.ListEarlyAccessRequests(ctx)
	if err != nil {
		return nil, err
	}

	for _, req := range rows {
		if req.Status == dbgen.EarlyAccessStatusPending && strings.EqualFold(req.Email, email) {
			return &req, nil
		}
	}

	return nil, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Service) List(ctx context.Context) ([]RequestResponse, error) {
	if err := s.authorizeProtectedAdmin(ctx); err != nil {
		return nil, err
	}

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
	if err := s.authorizeProtectedAdmin(ctx); err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	request, err := s.db.GetEarlyAccessRequest(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if request.Status != dbgen.EarlyAccessStatusPending {
		return nil, ErrAlreadyReviewed
	}

	err = s.sendApproval(ctx, request)
	if err != nil {
		return nil, err
	}

	updated, err := s.transitionPendingReview(ctx, id, dbgen.EarlyAccessStatusApproved)
	if err != nil {
		return nil, err
	}

	return toResponse(updated), nil
}

func (s *Service) Reject(ctx context.Context, id string) (*RequestResponse, error) {
	if err := s.authorizeProtectedAdmin(ctx); err != nil {
		return nil, err
	}

	updated, err := s.transitionPendingReview(ctx, id, dbgen.EarlyAccessStatusRejected)
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

var ErrInvalidToken = errors.New("invalid or expired action token")

func (s *Service) ApproveByToken(ctx context.Context, id, sig string, exp int64) (*RequestResponse, error) {
	if !validateActionToken(s.adminSecret, id, "approve", sig, exp) {
		return nil, ErrInvalidToken
	}
	return s.approveWithoutContextAuth(ctx, id)
}

func (s *Service) RejectByToken(ctx context.Context, id, sig string, exp int64) (*RequestResponse, error) {
	if !validateActionToken(s.adminSecret, id, "reject", sig, exp) {
		return nil, ErrInvalidToken
	}
	return s.rejectWithoutContextAuth(ctx, id)
}

func (s *Service) ValidateActionToken(id, action, sig string, exp int64) error {
	if !validateActionToken(s.adminSecret, id, action, sig, exp) {
		return ErrInvalidToken
	}
	return nil
}

func (s *Service) authorizeProtectedAdmin(ctx context.Context) error {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || !auth.IsAdminRole(identity.Role) {
		return ErrForbidden
	}
	if strings.TrimSpace(s.adminEmail) == "" {
		return ErrForbidden
	}

	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return ErrForbidden
	}
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(s.adminEmail)) {
		return ErrForbidden
	}

	return nil
}

func (s *Service) approveWithoutContextAuth(ctx context.Context, id string) (*RequestResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	request, err := s.db.GetEarlyAccessRequest(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if request.Status != dbgen.EarlyAccessStatusPending {
		return nil, ErrAlreadyReviewed
	}

	err = s.sendApproval(ctx, request)
	if err != nil {
		return nil, err
	}

	updated, err := s.transitionPendingReview(ctx, id, dbgen.EarlyAccessStatusApproved)
	if err != nil {
		return nil, err
	}

	return toResponse(updated), nil
}

func (s *Service) rejectWithoutContextAuth(ctx context.Context, id string) (*RequestResponse, error) {
	updated, err := s.transitionPendingReview(ctx, id, dbgen.EarlyAccessStatusRejected)
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func (s *Service) transitionPendingReview(ctx context.Context, id string, status dbgen.EarlyAccessStatus) (dbgen.EarlyAccessRequest, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return dbgen.EarlyAccessRequest{}, ErrNotFound
	}

	updated, err := s.db.UpdateEarlyAccessStatus(ctx, dbgen.UpdateEarlyAccessStatusParams{
		ID:     uid,
		Status: status,
	})
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return dbgen.EarlyAccessRequest{}, err
	}

	if _, getErr := s.db.GetEarlyAccessRequest(ctx, uid); getErr != nil {
		if errors.Is(getErr, pgx.ErrNoRows) {
			return dbgen.EarlyAccessRequest{}, ErrNotFound
		}
		return dbgen.EarlyAccessRequest{}, getErr
	}

	return dbgen.EarlyAccessRequest{}, ErrAlreadyReviewed
}

func (s *Service) sendApproval(ctx context.Context, req dbgen.EarlyAccessRequest) error {
	tempPass := make([]byte, 16)
	if _, readErr := rand.Read(tempPass); readErr != nil {
		return readErr
	}

	_, regErr := s.authService.Register(ctx, req.Email, hex.EncodeToString(tempPass), req.FullName)
	if regErr != nil && !errors.Is(regErr, auth.ErrEmailExists) {
		return regErr
	}

	setPasswordURL, err := s.authService.IssuePasswordResetURL(ctx, req.Email, s.appBaseURL)
	if err != nil {
		return err
	}

	_ = s.notifier.SendEarlyAccessApproval(ctx, req.Email, req.FullName, setPasswordURL)
	return nil
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
