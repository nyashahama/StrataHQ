package levy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/auth"
)

type fakeAttentionService struct {
	queueCalled   bool
	eventsCalled  bool
	recordCalled  bool
	recordInput   RecordCollectionEventInput
	queueResponse *AttentionQueueResponse
	events        []CollectionEvent
	err           error
}

func (f *fakeAttentionService) AttentionQueue(_ context.Context, _ auth.Identity, _ string) (*AttentionQueueResponse, error) {
	f.queueCalled = true
	if f.queueResponse != nil {
		return f.queueResponse, f.err
	}
	return &AttentionQueueResponse{Items: []AttentionItem{}, Scope: "scheme"}, f.err
}

func (f *fakeAttentionService) CollectionEvents(_ context.Context, _ auth.Identity, _, _ string) ([]CollectionEvent, error) {
	f.eventsCalled = true
	return f.events, f.err
}

func (f *fakeAttentionService) RecordCollectionEvent(_ context.Context, _ auth.Identity, _, _ string, input RecordCollectionEventInput) (*CollectionEvent, error) {
	f.recordCalled = true
	f.recordInput = input
	event := &CollectionEvent{EventType: input.EventType}
	return event, f.err
}

func (f *fakeAttentionService) Dashboard(_ context.Context, _ auth.Identity, _ string) (*DashboardResponse, error) {
	return &DashboardResponse{}, nil
}

func (f *fakeAttentionService) CreatePeriod(_ context.Context, _ auth.Identity, _ string, _ CreatePeriodInput) (*PeriodInfo, error) {
	return &PeriodInfo{}, nil
}

func (f *fakeAttentionService) Reconcile(_ context.Context, _ auth.Identity, _ string, _ []ReconcilePaymentInput) (*ReconcileResult, error) {
	return &ReconcileResult{}, nil
}

func TestAttentionQueueRequiresAuth(t *testing.T) {
	svc := &fakeAttentionService{}
	h := NewHandlerWithService(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/levies/scheme-1/attention", nil)
	w := httptest.NewRecorder()

	h.AttentionQueue(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRecordCollectionEventRejectsMissingPromiseDate(t *testing.T) {
	svc := &fakeAttentionService{}
	h := NewHandlerWithService(svc)

	body := []byte(`{"event_type":"promise_to_pay","promise_amount_cents":245000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/levies/scheme-1/accounts/account-1/events", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "user-1", "org-1", "trustee"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("schemeId", "scheme-1")
	rctx.URLParams.Add("accountId", "account-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.RecordCollectionEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if svc.recordCalled {
		t.Fatalf("service should not be called for invalid input")
	}
}
