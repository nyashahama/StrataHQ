package earlyaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type stubService struct {
	approveByTokenFn    func(id, sig string, exp int64) (*RequestResponse, error)
	rejectByTokenFn     func(id, sig string, exp int64) (*RequestResponse, error)
	approveByTokenCalls int
	rejectByTokenCalls  int
}

func (s *stubService) Submit(_ context.Context, _ SubmitParams) (*RequestResponse, error) {
	return nil, nil
}

func (s *stubService) List(_ context.Context) ([]RequestResponse, error) {
	return nil, nil
}

func (s *stubService) Approve(_ context.Context, _ string) (*RequestResponse, error) {
	return nil, nil
}

func (s *stubService) Reject(_ context.Context, _ string) (*RequestResponse, error) {
	return nil, nil
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
