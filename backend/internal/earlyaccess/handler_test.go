package earlyaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/auth"
)

type stubService struct {
	submitFn            func(params SubmitParams)
	approveFn           func(id string) (*RequestResponse, error)
	rejectFn            func(id string) (*RequestResponse, error)
	submitCalls         int
	approveByTokenFn    func(id, sig string, exp int64) (*RequestResponse, error)
	rejectByTokenFn     func(id, sig string, exp int64) (*RequestResponse, error)
	approveByTokenCalls int
	rejectByTokenCalls  int
}

func (s *stubService) Submit(_ context.Context, params SubmitParams) (*RequestResponse, error) {
	s.submitCalls++
	if s.submitFn != nil {
		s.submitFn(params)
	}
	return nil, nil
}

func (s *stubService) List(_ context.Context) ([]RequestResponse, error) {
	return nil, nil
}

func (s *stubService) Approve(_ context.Context, id string) (*RequestResponse, error) {
	if s.approveFn != nil {
		return s.approveFn(id)
	}
	return &RequestResponse{ID: id, Status: "approved"}, nil
}

func (s *stubService) Reject(_ context.Context, id string) (*RequestResponse, error) {
	if s.rejectFn != nil {
		return s.rejectFn(id)
	}
	return &RequestResponse{ID: id, Status: "rejected"}, nil
}

func (s *stubService) ApproveByToken(_ context.Context, id, sig string, exp int64) (*RequestResponse, error) {
	s.approveByTokenCalls++
	if s.approveByTokenFn != nil {
		return s.approveByTokenFn(id, sig, exp)
	}
	return &RequestResponse{ID: id, Status: "approved"}, nil
}

func (s *stubService) RejectByToken(_ context.Context, id, sig string, exp int64) (*RequestResponse, error) {
	s.rejectByTokenCalls++
	if s.rejectByTokenFn != nil {
		return s.rejectByTokenFn(id, sig, exp)
	}
	return &RequestResponse{ID: id, Status: "rejected"}, nil
}

func TestPublicRoutes_GetApproveDoesNotMutate(t *testing.T) {
	svc := &stubService{}
	router := NewHandler(svc).PublicRoutes()

	exp := time.Now().Add(15 * time.Minute).Unix()
	req := httptest.NewRequest(http.MethodGet, "/request-123/approve?exp="+strconv.FormatInt(exp, 10)+"&sig=test-signature", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	if svc.approveByTokenCalls != 0 {
		t.Fatalf("approveByTokenCalls=%d, want 0", svc.approveByTokenCalls)
	}
}

func TestPublicRoutes_PostApproveMutatesOnce(t *testing.T) {
	svc := &stubService{}
	router := NewHandler(svc).PublicRoutes()

	exp := time.Now().Add(15 * time.Minute).Unix()
	req := httptest.NewRequest(http.MethodPost, "/request-123/approve?exp="+strconv.FormatInt(exp, 10)+"&sig=test-signature", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	if svc.approveByTokenCalls != 1 {
		t.Fatalf("approveByTokenCalls=%d, want 1", svc.approveByTokenCalls)
	}
}

func TestPublicRoutes_ApprovePageHasNoInlineStyles(t *testing.T) {
	svc := &stubService{}
	router := NewHandler(svc).PublicRoutes()

	exp := time.Now().Add(15 * time.Minute).Unix()
	req := httptest.NewRequest(http.MethodGet, "/request-123/approve?exp="+strconv.FormatInt(exp, 10)+"&sig=test-signature", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if strings.Contains(w.Body.String(), "style=") {
		t.Fatalf("approve page should not render inline styles: %s", w.Body.String())
	}
}

func TestProtectedReviewRoutesReturnConflictForAlreadyReviewedRequests(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*Handler, http.ResponseWriter, *http.Request)
		service *stubService
	}{
		{
			name: "approve",
			handler: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.Approve(w, r)
			},
			service: &stubService{
				approveFn: func(id string) (*RequestResponse, error) {
					if id != "request-123" {
						t.Fatalf("approve id = %q, want request-123", id)
					}
					return nil, ErrAlreadyReviewed
				},
			},
		},
		{
			name: "reject",
			handler: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.Reject(w, r)
			},
			service: &stubService{
				rejectFn: func(id string) (*RequestResponse, error) {
					if id != "request-123" {
						t.Fatalf("reject id = %q, want request-123", id)
					}
					return nil, ErrAlreadyReviewed
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.service)
			req := protectedReviewRequest("request-123")
			w := httptest.NewRecorder()

			tt.handler(h, w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s, want 409", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "already reviewed") {
				t.Fatalf("body should explain already-reviewed state: %s", w.Body.String())
			}
		})
	}
}

func protectedReviewRequest(id string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/"+id, nil)
	ctx := auth.ContextWithIdentity(req.Context(), "user-1", "org-1", string(auth.RoleAdmin))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestPublicRoutes_RejectsInvalidEarlyAccessSubmission(t *testing.T) {
	svc := &stubService{}
	router := NewHandler(svc).PublicRoutes()

	t.Run("invalid email format", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"full_name":   "Person Example",
			"email":       "not-an-email",
			"scheme_name": "Green View Estate",
			"unit_count":  12,
		})
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", w.Code)
		}
		if svc.submitCalls != 0 {
			t.Fatalf("submitCalls=%d, want 0", svc.submitCalls)
		}
	})

	t.Run("too long fields are rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"full_name":   strings.Repeat("A", 121),
			"email":       "person@example.com",
			"scheme_name": strings.Repeat("B", 181),
			"unit_count":  12,
		})
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", w.Code)
		}
		if svc.submitCalls != 0 {
			t.Fatalf("submitCalls=%d, want 0", svc.submitCalls)
		}
	})

	t.Run("honeypot field triggers rejection", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"full_name":   "Person Example",
			"email":       "person@example.com",
			"scheme_name": "Green View Estate",
			"unit_count":  12,
			"website":     "https://spam.example",
		})
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", w.Code)
		}
		if svc.submitCalls != 0 {
			t.Fatalf("submitCalls=%d, want 0", svc.submitCalls)
		}
	})
}

func TestPublicRoutes_AcceptsValidEarlyAccessSubmission(t *testing.T) {
	svc := &stubService{}
	var captured SubmitParams
	svc.submitFn = func(params SubmitParams) {
		captured = params
	}
	router := NewHandler(svc).PublicRoutes()

	body, _ := json.Marshal(map[string]interface{}{
		"full_name":   "Person Example",
		"email":       "  Person@Example.COM  ",
		"scheme_name": "  Green View Estate  ",
		"unit_count":  12,
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d, want 201", w.Code)
	}
	if svc.submitCalls != 1 {
		t.Fatalf("submitCalls=%d, want 1", svc.submitCalls)
	}
	if captured.FullName != "Person Example" {
		t.Fatalf("fullName=%q, want trimmed value", captured.FullName)
	}
	if captured.Email != "person@example.com" {
		t.Fatalf("email=%q, want normalized lower-case value", captured.Email)
	}
	if captured.SchemeName != "Green View Estate" {
		t.Fatalf("schemeName=%q, want trimmed value", captured.SchemeName)
	}
}
