package levy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type attentionService interface {
	ReminderDraft(ctx context.Context, identity auth.Identity, schemeID, accountID string) (*ReminderDraftResponse, error)
	SendReminder(ctx context.Context, identity auth.Identity, schemeID, accountID string, input SendReminderInput) (*CollectionEvent, error)
	AttentionQueue(ctx context.Context, identity auth.Identity, schemeID string) (*AttentionQueueResponse, error)
	CollectionEvents(ctx context.Context, identity auth.Identity, schemeID, accountID string) ([]CollectionEvent, error)
	RecordCollectionEvent(ctx context.Context, identity auth.Identity, schemeID, accountID string, input RecordCollectionEventInput) (*CollectionEvent, error)
	Dashboard(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error)
	CreatePeriod(ctx context.Context, identity auth.Identity, schemeID string, input CreatePeriodInput) (*PeriodInfo, error)
	Reconcile(ctx context.Context, identity auth.Identity, schemeID string, payments []ReconcilePaymentInput) (*ReconcileResult, error)
	ImportBankStatement(ctx context.Context, identity auth.Identity, schemeID string, input BankStatementImportInput) (*BankStatementImportResponse, error)
	GetBankStatementImport(ctx context.Context, identity auth.Identity, schemeID, importID string) (*BankStatementImportDetails, error)
	ApplyBankStatementImport(ctx context.Context, identity auth.Identity, schemeID, importID string, manualMatches []BankStatementManualMatchInput) (*BankStatementImportResponse, error)
}

type Handler struct {
	service attentionService
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func NewHandlerWithService(svc attentionService) *Handler {
	return &Handler{service: svc}
}

type collectionEventRequest struct {
	EventType          string  `json:"event_type"`
	PromiseAmountCents *int64  `json:"promise_amount_cents"`
	PromiseDate        *string `json:"promise_date"`
	Note               *string `json:"note"`
}

type createPeriodRequest struct {
	Label       string `json:"label"`
	DueDate     string `json:"due_date"`
	AmountCents int64  `json:"amount_cents"`
}

//nolint:govet // Keep request DTO fields grouped by API meaning rather than field packing.
type reconcilePaymentRequest struct {
	AmountCents int64   `json:"amount_cents"`
	BankRef     *string `json:"bank_ref"`
	AccountID   string  `json:"account_id"`
	PaymentDate string  `json:"payment_date"`
	Reference   string  `json:"reference"`
}

type reconcileRequest struct {
	Payments []reconcilePaymentRequest `json:"payments"`
}

type reminderChannelRequest struct {
	Enabled bool   `json:"enabled"`
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body"`
}

type sendReminderRequest struct {
	Email    reminderChannelRequest `json:"email"`
	WhatsApp reminderChannelRequest `json:"whatsapp"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeLevyError(w, err, "failed to load levy dashboard")
		return
	}

	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) CreatePeriod(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req createPeriodRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	dueDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.DueDate))
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "due_date must be YYYY-MM-DD")
		return
	}

	created, err := h.service.CreatePeriod(r.Context(), identity, chi.URLParam(r, "schemeId"), CreatePeriodInput{
		Label:       strings.TrimSpace(req.Label),
		DueDate:     dueDate,
		AmountCents: req.AmountCents,
	})
	if err != nil {
		writeLevyError(w, err, "failed to create levy period")
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Reconcile(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req reconcileRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	payments := make([]ReconcilePaymentInput, 0, len(req.Payments))
	for _, payment := range req.Payments {
		paymentDate, err := time.Parse("2006-01-02", strings.TrimSpace(payment.PaymentDate))
		if err != nil {
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "payment_date must be YYYY-MM-DD")
			return
		}
		payments = append(payments, ReconcilePaymentInput{
			AccountID:   strings.TrimSpace(payment.AccountID),
			PaymentDate: paymentDate,
			Reference:   strings.TrimSpace(payment.Reference),
			BankRef:     normalizeOptionalString(payment.BankRef),
			AmountCents: payment.AmountCents,
		})
	}

	result, err := h.service.Reconcile(r.Context(), identity, chi.URLParam(r, "schemeId"), payments)
	if err != nil {
		writeLevyError(w, err, "failed to reconcile levy payments")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) AttentionQueue(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	queue, err := h.service.AttentionQueue(r.Context(), identity, chi.URLParam(r, "schemeId"))
	if err != nil {
		writeLevyError(w, err, "failed to load attention queue")
		return
	}
	response.JSON(w, http.StatusOK, queue)
}

func (h *Handler) PortfolioAttentionQueue(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	queue, err := h.service.AttentionQueue(r.Context(), identity, "")
	if err != nil {
		writeLevyError(w, err, "failed to load attention queue")
		return
	}
	response.JSON(w, http.StatusOK, queue)
}

func (h *Handler) CollectionEvents(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	accountID, err := accountIDFromRoute(r)
	if err != nil {
		writeLevyError(w, err, "invalid account id")
		return
	}
	items, err := h.service.CollectionEvents(r.Context(), identity, chi.URLParam(r, "schemeId"), accountID)
	if err != nil {
		writeLevyError(w, err, "failed to load collection events")
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) RecordCollectionEvent(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	var req collectionEventRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	var promiseDate *time.Time
	if req.PromiseDate != nil {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.PromiseDate))
		if err != nil {
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "promise_date must be YYYY-MM-DD")
			return
		}
		promiseDate = &parsed
	}
	if strings.TrimSpace(req.EventType) == "promise_to_pay" && (req.PromiseAmountCents == nil || promiseDate == nil) {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "promise_to_pay requires amount and date")
		return
	}
	accountID, err := accountIDFromRoute(r)
	if err != nil {
		writeLevyError(w, err, "invalid account id")
		return
	}
	created, err := h.service.RecordCollectionEvent(r.Context(), identity, chi.URLParam(r, "schemeId"), accountID, RecordCollectionEventInput{
		EventType:          strings.TrimSpace(req.EventType),
		Note:               normalizeOptionalString(req.Note),
		PromiseAmountCents: req.PromiseAmountCents,
		PromiseDate:        promiseDate,
	})
	if err != nil {
		writeLevyError(w, err, "failed to record collection event")
		return
	}
	response.JSON(w, http.StatusCreated, created)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (h *Handler) ReminderDraft(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	accountID, err := accountIDFromRoute(r)
	if err != nil {
		writeLevyError(w, err, "invalid account id")
		return
	}
	draft, err := h.service.ReminderDraft(r.Context(), identity, chi.URLParam(r, "schemeId"), accountID)
	if err != nil {
		writeLevyError(w, err, "failed to load reminder draft")
		return
	}
	response.JSON(w, http.StatusOK, draft)
}

func (h *Handler) SendReminder(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	var req sendReminderRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	input := normalizeSendReminderInput(SendReminderInput{
		Email:    ReminderChannelInput{Enabled: req.Email.Enabled, Subject: req.Email.Subject, Body: req.Email.Body},
		WhatsApp: ReminderChannelInput{Enabled: req.WhatsApp.Enabled, Body: req.WhatsApp.Body},
	})
	if err := validateSendReminderInput(input); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "enabled reminder channels require message content")
		return
	}

	accountID, err := accountIDFromRoute(r)
	if err != nil {
		writeLevyError(w, err, "invalid account id")
		return
	}
	event, err := h.service.SendReminder(r.Context(), identity, chi.URLParam(r, "schemeId"), accountID, input)
	if err != nil {
		writeLevyError(w, err, "failed to send reminder")
		return
	}
	response.JSON(w, http.StatusCreated, event)
}

func (h *Handler) ImportBankStatement(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid multipart form data")
		return
	}

	bank := strings.TrimSpace(r.FormValue("bank"))
	if bank == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "bank is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "file is required")
		return
	}
	defer file.Close()

	rawCSV, err := io.ReadAll(file)
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "failed to read uploaded file")
		return
	}

	import_, err := h.service.ImportBankStatement(r.Context(), identity, chi.URLParam(r, "schemeId"), BankStatementImportInput{
		BankName:         bank,
		OriginalFilename: header.Filename,
		RawCSV:           rawCSV,
	})
	if err != nil {
		writeLevyError(w, err, "failed to import bank statement")
		return
	}

	response.JSON(w, http.StatusAccepted, import_)
}

func (h *Handler) GetBankStatementImport(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	details, err := h.service.GetBankStatementImport(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "importId"))
	if err != nil {
		writeLevyError(w, err, "failed to load bank statement import")
		return
	}

	response.JSON(w, http.StatusOK, details)
}

func (h *Handler) ApplyBankStatementImport(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	type manualMatchRequest struct {
		RowID       string  `json:"row_id"`
		AccountID   string  `json:"account_id"`
		PaymentDate string  `json:"payment_date"`
		AmountCents int64   `json:"amount_cents"`
		Reference   string  `json:"reference"`
		BankRef     *string `json:"bank_ref"`
	}

	type applyRequest struct {
		ManualMatches []manualMatchRequest `json:"manual_matches"`
	}

	var req applyRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	matches := make([]BankStatementManualMatchInput, len(req.ManualMatches))
	//nolint:gosimple // manualMatchRequest is a distinct named type
	for i, m := range req.ManualMatches {
		matches[i] = BankStatementManualMatchInput{
			RowID:       m.RowID,
			AccountID:   m.AccountID,
			PaymentDate: m.PaymentDate,
			AmountCents: m.AmountCents,
			Reference:   m.Reference,
			BankRef:     m.BankRef,
		}
	}

	result, err := h.service.ApplyBankStatementImport(r.Context(), identity, chi.URLParam(r, "schemeId"), chi.URLParam(r, "importId"), matches)
	if err != nil {
		writeLevyError(w, err, "failed to apply bank statement import")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func accountIDFromRoute(r *http.Request) (string, error) {
	accountID := chi.URLParam(r, "accountId")
	if _, err := uuid.Parse(accountID); err != nil {
		return "", ErrInvalidInput
	}
	return accountID, nil
}

func writeLevyError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case ErrForbidden:
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
	case ErrNotFound:
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "levy resource not found")
	case ErrInvalidInput:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, fallback)
	}
}
