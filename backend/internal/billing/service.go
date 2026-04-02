package billing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotConfigured = errors.New("billing not configured")
)

const defaultPlanCode = "starter"

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type SubscriptionResponse struct {
	CustomerID        *string `json:"customer_id"`
	SubscriptionID    *string `json:"subscription_id"`
	CheckoutSessionID *string `json:"checkout_session_id"`
	CurrentPeriodEnd  *string `json:"current_period_end"`
	OrgID             string  `json:"org_id"`
	Provider          string  `json:"provider"`
	Status            string  `json:"status"`
	PlanCode          string  `json:"plan_code"`
	CancelAtPeriodEnd bool    `json:"cancel_at_period_end"`
	EntitlementActive bool    `json:"entitlement_active"`
	HasPortalAccess   bool    `json:"has_portal_access"`
}

type CheckoutResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
}

type PortalResponse struct {
	URL string `json:"url"`
}

type CheckoutSessionInput struct {
	OrgID         string
	OrgName       string
	CustomerID    *string
	CustomerEmail *string
	SuccessURL    string
	CancelURL     string
	PlanCode      string
}

type CheckoutSession struct {
	ID         string
	URL        string
	CustomerID *string
}

type WebhookEvent struct {
	Type              string
	OrgID             string
	CustomerID        string
	SubscriptionID    string
	CheckoutSessionID string
	Status            string
	PlanCode          string
	CurrentPeriodEnd  *time.Time
	CancelAtPeriodEnd bool
}

type Provider interface {
	CreateCheckoutSession(ctx context.Context, input CheckoutSessionInput) (*CheckoutSession, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
	ParseWebhook(payload []byte, signature string) (*WebhookEvent, error)
}

type Service struct {
	db         *database.Pool
	provider   Provider
	appBaseURL string
}

func NewService(db *database.Pool, provider Provider, appBaseURL string) *Service {
	return &Service{
		db:         db,
		provider:   provider,
		appBaseURL: appBaseURL,
	}
}

func (s *Service) GetSubscription(ctx context.Context, identity auth.Identity) (*SubscriptionResponse, error) {
	org, err := s.requireAdminOrg(ctx, identity)
	if err != nil {
		return nil, err
	}

	subscription, err := s.db.Q.GetOrgSubscription(ctx, org.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response := defaultSubscription(org.ID)
			return &response, nil
		}
		return nil, err
	}

	response := mapSubscription(subscription)
	return &response, nil
}

func (s *Service) CreateCheckoutSession(ctx context.Context, identity auth.Identity) (*CheckoutResponse, error) {
	org, err := s.requireAdminOrg(ctx, identity)
	if err != nil {
		return nil, err
	}
	if s.provider == nil {
		return nil, ErrNotConfigured
	}

	existing, _ := s.db.Q.GetOrgSubscription(ctx, org.ID)
	var customerID *string
	if existing.CustomerID.Valid {
		value := existing.CustomerID.String
		customerID = &value
	}

	var customerEmail *string
	if org.ContactEmail.Valid && org.ContactEmail.String != "" {
		value := org.ContactEmail.String
		customerEmail = &value
	} else {
		userID, parseErr := uuid.Parse(identity.UserID)
		if parseErr == nil {
			user, userErr := s.db.Q.GetUserByID(ctx, userID)
			if userErr == nil {
				customerEmail = &user.Email
			}
		}
	}

	session, err := s.provider.CreateCheckoutSession(ctx, CheckoutSessionInput{
		OrgID:         org.ID.String(),
		OrgName:       org.Name,
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		SuccessURL:    s.appBaseURL + "/agent/settings?section=billing&checkout=success",
		CancelURL:     s.appBaseURL + "/agent/settings?section=billing&checkout=cancelled",
		PlanCode:      defaultPlanCode,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.db.Q.UpsertOrgSubscription(ctx, dbgen.UpsertOrgSubscriptionParams{
		OrgID:             org.ID,
		Provider:          "stripe",
		Status:            pickString(existing.Status, "checkout_pending"),
		PlanCode:          pickString(existing.PlanCode, defaultPlanCode),
		CustomerID:        optionalText(ptrOr(session.CustomerID, existing.CustomerID)),
		SubscriptionID:    existing.SubscriptionID,
		CheckoutSessionID: optionalText(&session.ID),
		CurrentPeriodEnd:  existing.CurrentPeriodEnd,
		CancelAtPeriodEnd: existing.CancelAtPeriodEnd,
	})
	if err != nil {
		return nil, err
	}

	_ = updated
	return &CheckoutResponse{
		URL:       session.URL,
		SessionID: session.ID,
	}, nil
}

func (s *Service) CreatePortalSession(ctx context.Context, identity auth.Identity) (*PortalResponse, error) {
	org, err := s.requireAdminOrg(ctx, identity)
	if err != nil {
		return nil, err
	}
	if s.provider == nil {
		return nil, ErrNotConfigured
	}

	subscription, err := s.db.Q.GetOrgSubscription(ctx, org.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !subscription.CustomerID.Valid || subscription.CustomerID.String == "" {
		return nil, ErrInvalidInput
	}

	url, err := s.provider.CreatePortalSession(ctx, subscription.CustomerID.String, s.appBaseURL+"/agent/settings?section=billing")
	if err != nil {
		return nil, err
	}

	return &PortalResponse{URL: url}, nil
}

func (s *Service) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.provider == nil {
		return ErrNotConfigured
	}

	event, err := s.provider.ParseWebhook(payload, signature)
	if err != nil {
		return err
	}
	if event.OrgID == "" && event.CustomerID == "" && event.SubscriptionID == "" {
		return nil
	}

	orgID, err := s.resolveWebhookOrgID(ctx, event)
	if err != nil {
		return err
	}

	existing, err := s.db.Q.GetOrgSubscription(ctx, orgID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	status := pickString(event.Status, pickString(existing.Status, "inactive"))
	planCode := pickString(event.PlanCode, pickString(existing.PlanCode, defaultPlanCode))

	_, err = s.db.Q.UpsertOrgSubscription(ctx, dbgen.UpsertOrgSubscriptionParams{
		OrgID:             orgID,
		Provider:          "stripe",
		Status:            status,
		PlanCode:          planCode,
		CustomerID:        mergeText(existing.CustomerID, event.CustomerID),
		SubscriptionID:    mergeText(existing.SubscriptionID, event.SubscriptionID),
		CheckoutSessionID: mergeText(existing.CheckoutSessionID, event.CheckoutSessionID),
		CurrentPeriodEnd:  mergeTime(existing.CurrentPeriodEnd, event.CurrentPeriodEnd),
		CancelAtPeriodEnd: event.CancelAtPeriodEnd,
	})
	return err
}

func (s *Service) requireAdminOrg(ctx context.Context, identity auth.Identity) (dbgen.Org, error) {
	if !auth.IsAdminRole(identity.Role) {
		return dbgen.Org{}, ErrForbidden
	}

	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return dbgen.Org{}, ErrInvalidInput
	}

	org, err := s.db.Q.GetOrg(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Org{}, ErrNotFound
		}
		return dbgen.Org{}, err
	}
	return org, nil
}

func (s *Service) resolveWebhookOrgID(ctx context.Context, event *WebhookEvent) (uuid.UUID, error) {
	if event.OrgID != "" {
		return uuid.Parse(event.OrgID)
	}
	if event.SubscriptionID != "" {
		subscription, err := s.db.Q.GetOrgSubscriptionBySubscriptionID(ctx, pgtype.Text{String: event.SubscriptionID, Valid: true})
		if err == nil {
			return subscription.OrgID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, err
		}
	}
	if event.CustomerID != "" {
		subscription, err := s.db.Q.GetOrgSubscriptionByCustomerID(ctx, pgtype.Text{String: event.CustomerID, Valid: true})
		if err == nil {
			return subscription.OrgID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, err
		}
	}
	return uuid.UUID{}, ErrInvalidInput
}

func defaultSubscription(orgID uuid.UUID) SubscriptionResponse {
	return SubscriptionResponse{
		OrgID:             orgID.String(),
		Provider:          "stripe",
		Status:            "inactive",
		PlanCode:          defaultPlanCode,
		CancelAtPeriodEnd: false,
		EntitlementActive: false,
		HasPortalAccess:   false,
	}
}

func mapSubscription(subscription dbgen.OrgSubscription) SubscriptionResponse {
	return SubscriptionResponse{
		CustomerID:        textPointer(subscription.CustomerID),
		SubscriptionID:    textPointer(subscription.SubscriptionID),
		CheckoutSessionID: textPointer(subscription.CheckoutSessionID),
		CurrentPeriodEnd:  timePointer(subscription.CurrentPeriodEnd),
		OrgID:             subscription.OrgID.String(),
		Provider:          subscription.Provider,
		Status:            subscription.Status,
		PlanCode:          subscription.PlanCode,
		CancelAtPeriodEnd: subscription.CancelAtPeriodEnd,
		EntitlementActive: entitlementActive(subscription.Status),
		HasPortalAccess:   subscription.CustomerID.Valid && subscription.CustomerID.String != "",
	}
}

func entitlementActive(status string) bool {
	return status == "active" || status == "trialing"
}

func optionalText(value *string) pgtype.Text {
	if value == nil || *value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	copy := value.String
	return &copy
}

func timePointer(value pgtype.Timestamptz) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}

func mergeText(existing pgtype.Text, next string) pgtype.Text {
	if next != "" {
		return pgtype.Text{String: next, Valid: true}
	}
	return existing
}

func mergeTime(existing pgtype.Timestamptz, next *time.Time) pgtype.Timestamptz {
	if next == nil {
		return existing
	}
	return pgtype.Timestamptz{Time: *next, Valid: true}
}

func pickString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func ptrOr(primary *string, fallback pgtype.Text) *string {
	if primary != nil && *primary != "" {
		return primary
	}
	if fallback.Valid && fallback.String != "" {
		copy := fallback.String
		return &copy
	}
	return nil
}
