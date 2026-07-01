package integrations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

var defaultScopes = []string{"read:schemes", "read:levies", "read:financials"}

type APIClientInfo struct {
	ExpiresAt  *time.Time `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ID         string     `json:"id"`
	OrgID      string     `json:"org_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	SchemeIDs  []string   `json:"scheme_ids"`
	CreatedAt  time.Time  `json:"created_at"`
}

type APIClientCreateResponse struct {
	APIClientInfo
	APIKey string `json:"api_key"`
}

type CreateAPIClientInput struct {
	Name      string
	SchemeIDs []string
	Scopes    []string
	ExpiresAt *time.Time
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) CreateAPIClient(ctx context.Context, identity auth.Identity, input CreateAPIClientInput) (*APIClientCreateResponse, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if input.Name == "" || len(input.SchemeIDs) == 0 {
		return nil, ErrInvalidInput
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return nil, ErrInvalidInput
	}
	scopeValues := input.Scopes
	if len(scopeValues) == 0 {
		scopeValues = defaultScopes
	}
	if !validScopes(scopeValues) {
		return nil, ErrInvalidInput
	}
	schemeUUIDs, err := parseUUIDs(input.SchemeIDs)
	if err != nil {
		return nil, ErrInvalidInput
	}
	schemeUUIDs = dedupUUIDs(schemeUUIDs)
	count, err := s.db.Q.CountSchemesByOrgAndIDs(ctx, dbgen.CountSchemesByOrgAndIDsParams{OrgID: orgID, SchemeIds: schemeUUIDs})
	if err != nil {
		return nil, err
	}
	if count != int32(len(schemeUUIDs)) {
		return nil, ErrForbidden
	}
	generated, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	var expiresAt pgtype.Timestamptz
	if input.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *input.ExpiresAt, Valid: true}
	}
	created, err := s.db.Q.CreateIntegrationAPIClient(ctx, dbgen.CreateIntegrationAPIClientParams{
		OrgID:           orgID,
		Name:            input.Name,
		KeyPrefix:       generated.Prefix,
		KeyHash:         generated.Hash,
		Scopes:          scopeValues,
		CreatedByUserID: pgtype.UUID{Bytes: userID, Valid: true},
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		return nil, err
	}
	for _, schemeID := range schemeUUIDs {
		if linkErr := s.db.Q.LinkIntegrationAPIClientScheme(ctx, dbgen.LinkIntegrationAPIClientSchemeParams{ClientID: created.ID, SchemeID: schemeID}); linkErr != nil {
			return nil, linkErr
		}
	}
	info, err := s.mapClient(ctx, created)
	if err != nil {
		return nil, err
	}
	return &APIClientCreateResponse{APIClientInfo: info, APIKey: generated.Raw}, nil
}

func (s *Service) ListAPIClients(ctx context.Context, identity auth.Identity) ([]APIClientInfo, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.Q.ListIntegrationAPIClientsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]APIClientInfo, 0, len(rows))
	for _, row := range rows {
		mapped, mapErr := s.mapClient(ctx, row)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (s *Service) RevokeAPIClient(ctx context.Context, identity auth.Identity, clientID string) (*APIClientInfo, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	cid, err := uuid.Parse(clientID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.db.Q.RevokeIntegrationAPIClient(ctx, dbgen.RevokeIntegrationAPIClientParams{ID: cid, OrgID: orgID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	info, err := s.mapClient(ctx, row)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func validScopes(scopes []string) bool {
	allowed := map[string]struct{}{"read:schemes": {}, "read:levies": {}, "read:financials": {}}
	for _, scope := range scopes {
		if _, ok := allowed[scope]; !ok {
			return false
		}
	}
	return true
}

func dedupUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) mapClient(ctx context.Context, row dbgen.IntegrationApiClient) (APIClientInfo, error) {
	schemes, err := s.db.Q.ListIntegrationAPIClientSchemes(ctx, row.ID)
	if err != nil {
		return APIClientInfo{}, err
	}
	schemeIDs := make([]string, 0, len(schemes))
	for _, schemeID := range schemes {
		schemeIDs = append(schemeIDs, schemeID.String())
	}
	return APIClientInfo{
		ID:         row.ID.String(),
		OrgID:      row.OrgID.String(),
		Name:       row.Name,
		KeyPrefix:  row.KeyPrefix,
		Scopes:     row.Scopes,
		SchemeIDs:  schemeIDs,
		ExpiresAt:  timestamptzPtr(row.ExpiresAt),
		RevokedAt:  timestamptzPtr(row.RevokedAt),
		LastUsedAt: timestamptzPtr(row.LastUsedAt),
		CreatedAt:  row.CreatedAt,
	}, nil
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, raw string) (Identity, error) {
	prefix, err := ParseAPIKeyPrefix(raw)
	if err != nil {
		return Identity{}, ErrForbidden
	}
	client, err := s.db.Q.GetIntegrationAPIClientByPrefix(ctx, prefix)
	if err != nil {
		return Identity{}, ErrForbidden
	}
	now := time.Now()
	if client.RevokedAt.Valid || (client.ExpiresAt.Valid && client.ExpiresAt.Time.Before(now)) {
		return Identity{}, ErrForbidden
	}
	if !CompareAPIKeyHash(raw, client.KeyHash) {
		return Identity{}, ErrForbidden
	}
	schemes, err := s.db.Q.ListIntegrationAPIClientSchemes(ctx, client.ID)
	if err != nil {
		return Identity{}, err
	}
	identity := Identity{
		ClientID:  client.ID.String(),
		OrgID:     client.OrgID.String(),
		Scopes:    client.Scopes,
		SchemeIDs: make([]string, 0, len(schemes)),
	}
	for _, schemeID := range schemes {
		identity.SchemeIDs = append(identity.SchemeIDs, schemeID.String())
	}
	_ = s.db.Q.TouchIntegrationAPIClientLastUsed(ctx, client.ID)
	return identity, nil
}

type OpenAPISchemeInfo struct {
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UnitCount int       `json:"unit_count"`
}

type OpenAPIUnitInfo struct {
	CreatedAt       time.Time `json:"created_at"`
	ID              string    `json:"id"`
	SchemeID        string    `json:"scheme_id"`
	Identifier      string    `json:"identifier"`
	SectionValueBps int32     `json:"section_value_bps"`
}

type OpenAPILevyPeriodInfo struct {
	CreatedAt   time.Time `json:"created_at"`
	DueDate     time.Time `json:"due_date"`
	ID          string    `json:"id"`
	SchemeID    string    `json:"scheme_id"`
	Label       string    `json:"label"`
	AmountCents int64     `json:"amount_cents"`
}

type OpenAPILevyAccountInfo struct {
	UpdatedAt      time.Time `json:"updated_at"`
	ID             string    `json:"id"`
	SchemeID       string    `json:"scheme_id"`
	UnitID         string    `json:"unit_id"`
	UnitIdentifier string    `json:"unit_identifier"`
	PeriodID       string    `json:"period_id"`
	PeriodLabel    string    `json:"period_label"`
	AmountCents    int64     `json:"amount_cents"`
	PaidCents      int64     `json:"paid_cents"`
	Status         string    `json:"status"`
}

type OpenAPILevyPaymentInfo struct {
	PaymentDate    time.Time `json:"payment_date"`
	CreatedAt      time.Time `json:"created_at"`
	ID             string    `json:"id"`
	SchemeID       string    `json:"scheme_id"`
	LevyAccountID  string    `json:"levy_account_id"`
	UnitID         string    `json:"unit_id"`
	UnitIdentifier string    `json:"unit_identifier"`
	AmountCents    int64     `json:"amount_cents"`
	Reference      string    `json:"reference"`
	BankRef        string    `json:"bank_ref"`
}

type OpenAPILevyAccountFilters struct {
	PeriodID     string
	Status       string
	UpdatedSince string
	LimitRows    int32
	OffsetRows   int32
}

type OpenAPILevyPaymentFilters struct {
	FromDate   string
	ToDate     string
	LimitRows  int32
	OffsetRows int32
}

func (s *Service) ListOpenAPISchemes(ctx context.Context, identity Identity) ([]OpenAPISchemeInfo, error) {
	clientID, err := uuid.Parse(identity.ClientID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.Q.ListOpenAPISchemesByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	result := make([]OpenAPISchemeInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, OpenAPISchemeInfo{
			ID:        row.ID.String(),
			Name:      row.Name,
			Address:   row.Address,
			UnitCount: int(row.UnitCount),
			CreatedAt: row.CreatedAt,
		})
	}
	return result, nil
}

func (s *Service) GetOpenAPIScheme(ctx context.Context, identity Identity, schemeID string) (*OpenAPISchemeInfo, error) {
	clientID, err := uuid.Parse(identity.ClientID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.db.Q.GetOpenAPISchemeByClient(ctx, dbgen.GetOpenAPISchemeByClientParams{ClientID: clientID, SchemeID: sid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &OpenAPISchemeInfo{
		ID:        row.ID.String(),
		Name:      row.Name,
		Address:   row.Address,
		UnitCount: int(row.UnitCount),
		CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Service) ListOpenAPIUnits(ctx context.Context, schemeID string) ([]OpenAPIUnitInfo, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.Q.ListOpenAPIUnitsByScheme(ctx, sid)
	if err != nil {
		return nil, err
	}
	result := make([]OpenAPIUnitInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, OpenAPIUnitInfo{
			ID:              row.ID.String(),
			SchemeID:        row.SchemeID.String(),
			Identifier:      row.Identifier,
			SectionValueBps: row.SectionValueBps,
			CreatedAt:       row.CreatedAt,
		})
	}
	return result, nil
}

func (s *Service) ListOpenAPILevyPeriods(ctx context.Context, schemeID string) ([]OpenAPILevyPeriodInfo, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.Q.ListOpenAPILevyPeriodsByScheme(ctx, sid)
	if err != nil {
		return nil, err
	}
	result := make([]OpenAPILevyPeriodInfo, 0, len(rows))
	for _, row := range rows {
		var dueDate time.Time
		if row.DueDate.Valid {
			dueDate = row.DueDate.Time
		}
		result = append(result, OpenAPILevyPeriodInfo{
			ID:          row.ID.String(),
			SchemeID:    row.SchemeID.String(),
			Label:       row.Label,
			AmountCents: row.AmountCents,
			DueDate:     dueDate,
			CreatedAt:   row.CreatedAt,
		})
	}
	return result, nil
}

func (s *Service) ListOpenAPILevyAccounts(ctx context.Context, schemeID string, filters OpenAPILevyAccountFilters) ([]OpenAPILevyAccountInfo, int, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, 0, ErrInvalidInput
	}
	var periodID pgtype.UUID
	if filters.PeriodID != "" {
		pid, parseErr := uuid.Parse(filters.PeriodID)
		if parseErr != nil {
			return nil, 0, ErrInvalidInput
		}
		periodID = pgtype.UUID{Bytes: pid, Valid: true}
	}
	var status pgtype.Text
	if filters.Status != "" {
		filters.Status = strings.TrimSpace(filters.Status)
		if !validOpenAPILevyAccountStatus(filters.Status) {
			return nil, 0, ErrInvalidInput
		}
		status = pgtype.Text{String: filters.Status, Valid: true}
	}
	var updatedSince pgtype.Timestamptz
	if filters.UpdatedSince != "" {
		ts, parseErr := time.Parse(time.RFC3339, filters.UpdatedSince)
		if parseErr != nil {
			return nil, 0, ErrInvalidInput
		}
		updatedSince = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	countParams := dbgen.CountOpenAPILevyAccountsBySchemeParams{
		SchemeID:     sid,
		PeriodID:     periodID,
		Status:       status,
		UpdatedSince: updatedSince,
	}
	total, err := s.db.Q.CountOpenAPILevyAccountsByScheme(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	listParams := dbgen.ListOpenAPILevyAccountsBySchemeParams{
		SchemeID:     sid,
		PeriodID:     periodID,
		Status:       status,
		UpdatedSince: updatedSince,
		LimitRows:    filters.LimitRows,
		OffsetRows:   filters.OffsetRows,
	}
	rows, err := s.db.Q.ListOpenAPILevyAccountsByScheme(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	result := make([]OpenAPILevyAccountInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, OpenAPILevyAccountInfo{
			ID:             row.ID.String(),
			SchemeID:       row.SchemeID.String(),
			UnitID:         row.UnitID.String(),
			UnitIdentifier: row.UnitIdentifier,
			PeriodID:       row.PeriodID.String(),
			PeriodLabel:    row.PeriodLabel,
			AmountCents:    row.AmountCents,
			PaidCents:      row.PaidCents,
			Status:         row.Status,
			UpdatedAt:      row.UpdatedAt,
		})
	}
	return result, int(total), nil
}

func validOpenAPILevyAccountStatus(status string) bool {
	switch status {
	case "paid", "partial", "overdue", "pending":
		return true
	default:
		return false
	}
}

func (s *Service) ListOpenAPILevyPayments(ctx context.Context, schemeID string, filters OpenAPILevyPaymentFilters) ([]OpenAPILevyPaymentInfo, int, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, 0, ErrInvalidInput
	}
	var fromDate, toDate pgtype.Date
	if filters.FromDate != "" {
		ts, parseErr := time.Parse("2006-01-02", filters.FromDate)
		if parseErr != nil {
			return nil, 0, ErrInvalidInput
		}
		fromDate = pgtype.Date{Time: ts, Valid: true}
	}
	if filters.ToDate != "" {
		ts, parseErr := time.Parse("2006-01-02", filters.ToDate)
		if parseErr != nil {
			return nil, 0, ErrInvalidInput
		}
		toDate = pgtype.Date{Time: ts, Valid: true}
	}
	countParams := dbgen.CountOpenAPILevyPaymentsBySchemeParams{
		SchemeID: sid,
		FromDate: fromDate,
		ToDate:   toDate,
	}
	total, err := s.db.Q.CountOpenAPILevyPaymentsByScheme(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	listParams := dbgen.ListOpenAPILevyPaymentsBySchemeParams{
		SchemeID:   sid,
		FromDate:   fromDate,
		ToDate:     toDate,
		LimitRows:  filters.LimitRows,
		OffsetRows: filters.OffsetRows,
	}
	rows, err := s.db.Q.ListOpenAPILevyPaymentsByScheme(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	result := make([]OpenAPILevyPaymentInfo, 0, len(rows))
	for _, row := range rows {
		var paymentDate time.Time
		if row.PaymentDate.Valid {
			paymentDate = row.PaymentDate.Time
		}
		var bankRef string
		if row.BankRef.Valid {
			bankRef = row.BankRef.String
		}
		result = append(result, OpenAPILevyPaymentInfo{
			ID:             row.ID.String(),
			SchemeID:       row.SchemeID.String(),
			LevyAccountID:  row.LevyAccountID.String(),
			UnitID:         row.UnitID.String(),
			UnitIdentifier: row.UnitIdentifier,
			AmountCents:    row.AmountCents,
			PaymentDate:    paymentDate,
			Reference:      row.Reference,
			BankRef:        bankRef,
			CreatedAt:      row.CreatedAt,
		})
	}
	return result, int(total), nil
}
