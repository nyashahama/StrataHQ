package compliance

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
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

//nolint:govet // Keep response fields grouped by domain meaning rather than field packing.
type ItemInfo struct {
	DueDate     *string   `json:"due_date"`
	ID          string    `json:"id"`
	SchemeID    string    `json:"scheme_id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Requirement string    `json:"requirement"`
	Status      string    `json:"status"`
	Detail      string    `json:"detail"`
	Action      string    `json:"action"`
	AssessedAt  time.Time `json:"assessed_at"`
}

//nolint:govet // Keep response fields grouped by domain meaning rather than field packing.
type DashboardResponse struct {
	Items             []ItemInfo `json:"items"`
	Role              string     `json:"role"`
	Score             int        `json:"score"`
	Total             int        `json:"total"`
	CompliantCount    int        `json:"compliant_count"`
	AtRiskCount       int        `json:"at_risk_count"`
	NonCompliantCount int        `json:"non_compliant_count"`
	LastAssessedAt    time.Time  `json:"last_assessed_at"`
	UpcomingDeadlines int        `json:"upcoming_deadlines"`
}

type Service struct {
	db                   *database.Pool
	listSchemesByOrgFn   func(context.Context, uuid.UUID) ([]dbgen.Scheme, error)
	dashboardForSchemeFn func(context.Context, uuid.UUID) (*DashboardResponse, error)
}

func NewService(db *database.Pool) *Service {
	svc := &Service{db: db}
	svc.dashboardForSchemeFn = svc.dashboardForScheme
	return svc
}

func (s *Service) Dashboard(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(role) {
		return nil, ErrForbidden
	}

	rows, err := s.db.Q.ListComplianceItemsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	resp := &DashboardResponse{
		Items:          make([]ItemInfo, 0, len(rows)),
		Role:           role,
		LastAssessedAt: time.Time{},
	}

	totalPoints := 0
	earnedPoints := 0

	for _, row := range rows {
		item := ItemInfo{
			ID:          row.ID.String(),
			SchemeID:    row.SchemeID.String(),
			Category:    string(row.Category),
			Title:       row.Title,
			Requirement: row.Requirement,
			Status:      string(row.Status),
			Detail:      row.Detail,
			Action:      row.Action,
			AssessedAt:  row.AssessedAt,
		}
		if row.DueDate.Valid {
			date := row.DueDate.Time.Format("2006-01-02")
			item.DueDate = &date
		}

		resp.Items = append(resp.Items, item)
		resp.Total++
		totalPoints += 10
		earnedPoints += statusPoints(row.Status)

		switch row.Status {
		case dbgen.ComplianceStatusCompliant:
			resp.CompliantCount++
		case dbgen.ComplianceStatusAtRisk:
			resp.AtRiskCount++
		case dbgen.ComplianceStatusNonCompliant:
			resp.NonCompliantCount++
		}

		if row.AssessedAt.After(resp.LastAssessedAt) {
			resp.LastAssessedAt = row.AssessedAt
		}
	}

	if totalPoints > 0 {
		resp.Score = int(float64(earnedPoints) / float64(totalPoints) * 100)
	}

	deadlines, err := s.db.Q.CountUpcomingDeadlinesByScheme(ctx, scheme.ID)
	if err == nil {
		resp.UpcomingDeadlines = int(deadlines)
	}

	return resp, nil
}

func (s *Service) resolveSchemeAccess(ctx context.Context, identity auth.Identity, schemeID string) (dbgen.Scheme, string, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return dbgen.Scheme{}, "", ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", ErrNotFound
		}
		return dbgen.Scheme{}, "", err
	}

	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return dbgen.Scheme{}, "", ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return dbgen.Scheme{}, "", ErrForbidden
		}
		return scheme, string(auth.RoleAdmin), nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return dbgen.Scheme{}, "", ErrInvalidInput
	}

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", ErrForbidden
		}
		return dbgen.Scheme{}, "", err
	}

	return scheme, membership.Role, nil
}

func statusPoints(status dbgen.ComplianceStatus) int {
	switch status {
	case dbgen.ComplianceStatusCompliant:
		return 10
	case dbgen.ComplianceStatusAtRisk:
		return 5
	default:
		return 0
	}
}

type CreateItemInput struct {
	Category    string     `json:"category"`
	Title       string     `json:"title"`
	Requirement string     `json:"requirement"`
	Detail      string     `json:"detail"`
	Action      string     `json:"action"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateItemInput struct {
	Status  *string    `json:"status,omitempty"`
	Detail  *string    `json:"detail,omitempty"`
	Action  *string    `json:"action,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

func (s *Service) CreateItem(ctx context.Context, identity auth.Identity, schemeID string, input CreateItemInput) (*ItemInfo, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(role) {
		return nil, ErrForbidden
	}

	input.Category = strings.TrimSpace(input.Category)
	input.Title = strings.TrimSpace(input.Title)
	input.Requirement = strings.TrimSpace(input.Requirement)
	input.Detail = strings.TrimSpace(input.Detail)
	input.Action = strings.TrimSpace(input.Action)
	if !validCategory(input.Category) || input.Title == "" || input.Requirement == "" || input.Detail == "" || input.Action == "" {
		return nil, ErrInvalidInput
	}

	var dueDate pgtype.Date
	if input.DueDate != nil {
		dueDate = pgtype.Date{Time: *input.DueDate, Valid: true}
	}

	item, err := s.db.Q.CreateComplianceItem(ctx, dbgen.CreateComplianceItemParams{
		SchemeID:    scheme.ID,
		Category:    dbgen.ComplianceCategory(input.Category),
		Title:       input.Title,
		Requirement: input.Requirement,
		Status:      dbgen.ComplianceStatusNonCompliant,
		Detail:      input.Detail,
		Action:      input.Action,
		DueDate:     dueDate,
		AssessedAt:  time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	result := &ItemInfo{
		ID:          item.ID.String(),
		SchemeID:    item.SchemeID.String(),
		Category:    string(item.Category),
		Title:       item.Title,
		Requirement: item.Requirement,
		Status:      string(item.Status),
		Detail:      item.Detail,
		Action:      item.Action,
		AssessedAt:  item.AssessedAt,
	}
	if item.DueDate.Valid {
		date := item.DueDate.Time.Format("2006-01-02")
		result.DueDate = &date
	}
	return result, nil
}

func (s *Service) UpdateItem(ctx context.Context, identity auth.Identity, schemeID, itemID string, input UpdateItemInput) (*ItemInfo, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(role) {
		return nil, ErrForbidden
	}

	id, err := uuid.Parse(itemID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	existing, err := s.db.Q.GetComplianceItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if existing.SchemeID != scheme.ID {
		return nil, ErrForbidden
	}

	status := existing.Status
	if input.Status != nil && validStatus(*input.Status) {
		status = dbgen.ComplianceStatus(*input.Status)
	}
	detail := existing.Detail
	if input.Detail != nil {
		detail = *input.Detail
	}
	action := existing.Action
	if input.Action != nil {
		action = *input.Action
	}
	dueDate := existing.DueDate
	if input.DueDate != nil {
		dueDate = pgtype.Date{Time: *input.DueDate, Valid: true}
	}

	updated, err := s.db.Q.UpdateComplianceItem(ctx, dbgen.UpdateComplianceItemParams{
		ID:      id,
		Status:  status,
		Detail:  detail,
		Action:  action,
		DueDate: dueDate,
	})
	if err != nil {
		return nil, err
	}

	result := mapItem(updated)
	return &result, nil
}

func (s *Service) DeleteItem(ctx context.Context, identity auth.Identity, schemeID, itemID string) error {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return err
	}
	if auth.IsResidentRole(role) {
		return ErrForbidden
	}

	id, err := uuid.Parse(itemID)
	if err != nil {
		return ErrInvalidInput
	}

	existing, err := s.db.Q.GetComplianceItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if existing.SchemeID != scheme.ID {
		return ErrForbidden
	}

	return s.db.Q.DeleteComplianceItem(ctx, id)
}

func (s *Service) Assess(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
	dashboard, err := s.Dashboard(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	scheme, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.Q.CreateComplianceAssessment(ctx, dbgen.CreateComplianceAssessmentParams{
		SchemeID:          scheme.ID,
		Score:             int32(dashboard.Score),
		TotalItems:        int32(dashboard.Total),
		CompliantCount:    int32(dashboard.CompliantCount),
		AtRiskCount:       int32(dashboard.AtRiskCount),
		NonCompliantCount: int32(dashboard.NonCompliantCount),
	})

	return dashboard, nil
}

type PortfolioSchemeInfo struct {
	SchemeID          string    `json:"scheme_id"`
	SchemeName        string    `json:"scheme_name"`
	Score             int       `json:"score"`
	Total             int       `json:"total"`
	CompliantCount    int       `json:"compliant_count"`
	AtRiskCount       int       `json:"at_risk_count"`
	NonCompliantCount int       `json:"non_compliant_count"`
	UpcomingDeadlines int       `json:"upcoming_deadlines"`
	LastAssessedAt    time.Time `json:"last_assessed_at"`
}

type PortfolioDashboardResponse struct {
	Schemes         []PortfolioSchemeInfo `json:"schemes"`
	OverallScore    int                   `json:"overall_score"`
	TotalSchemes    int                   `json:"total_schemes"`
	HealthySchemes  int                   `json:"healthy_schemes"`
	AtRiskSchemes   int                   `json:"at_risk_schemes"`
	CriticalSchemes int                   `json:"critical_schemes"`
}

func (s *Service) PortfolioDashboard(ctx context.Context, identity auth.Identity) (*PortfolioDashboardResponse, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}

	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	listSchemesByOrg := s.listSchemesByOrgFn
	if listSchemesByOrg == nil {
		listSchemesByOrg = s.db.Q.ListSchemesByOrg
	}
	rows, err := listSchemesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp := &PortfolioDashboardResponse{
		Schemes: make([]PortfolioSchemeInfo, 0, len(rows)),
	}
	var totalScore int

	for _, row := range rows {
		dashboardForScheme := s.dashboardForSchemeFn
		if dashboardForScheme == nil {
			dashboardForScheme = s.dashboardForScheme
		}
		dashboard, dashErr := dashboardForScheme(ctx, row.ID)
		if dashErr != nil {
			return nil, dashErr
		}

		info := PortfolioSchemeInfo{
			SchemeID:          row.ID.String(),
			SchemeName:        row.Name,
			Score:             dashboard.Score,
			Total:             dashboard.Total,
			CompliantCount:    dashboard.CompliantCount,
			AtRiskCount:       dashboard.AtRiskCount,
			NonCompliantCount: dashboard.NonCompliantCount,
			UpcomingDeadlines: dashboard.UpcomingDeadlines,
			LastAssessedAt:    dashboard.LastAssessedAt,
		}
		resp.Schemes = append(resp.Schemes, info)
		totalScore += dashboard.Score

		switch {
		case dashboard.Score >= 80:
			resp.HealthySchemes++
		case dashboard.Score >= 60:
			resp.AtRiskSchemes++
		default:
			resp.CriticalSchemes++
		}
	}

	resp.TotalSchemes = len(resp.Schemes)
	if resp.TotalSchemes > 0 {
		resp.OverallScore = totalScore / resp.TotalSchemes
	}

	return resp, nil
}

func (s *Service) dashboardForScheme(ctx context.Context, schemeID uuid.UUID) (*DashboardResponse, error) {
	rows, err := s.db.Q.ListComplianceItemsByScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}

	resp := &DashboardResponse{
		Items: make([]ItemInfo, 0, len(rows)),
	}

	totalPoints := 0
	earnedPoints := 0

	for _, row := range rows {
		item := mapItem(row)
		resp.Items = append(resp.Items, item)
		resp.Total++
		totalPoints += 10
		earnedPoints += statusPoints(row.Status)

		switch row.Status {
		case dbgen.ComplianceStatusCompliant:
			resp.CompliantCount++
		case dbgen.ComplianceStatusAtRisk:
			resp.AtRiskCount++
		case dbgen.ComplianceStatusNonCompliant:
			resp.NonCompliantCount++
		}

		if row.AssessedAt.After(resp.LastAssessedAt) {
			resp.LastAssessedAt = row.AssessedAt
		}
	}

	if totalPoints > 0 {
		resp.Score = int(float64(earnedPoints) / float64(totalPoints) * 100)
	}

	deadlines, _ := s.db.Q.CountUpcomingDeadlinesByScheme(ctx, schemeID)
	resp.UpcomingDeadlines = int(deadlines)

	return resp, nil
}

func mapItem(row dbgen.ComplianceItem) ItemInfo {
	item := ItemInfo{
		ID:          row.ID.String(),
		SchemeID:    row.SchemeID.String(),
		Category:    string(row.Category),
		Title:       row.Title,
		Requirement: row.Requirement,
		Status:      string(row.Status),
		Detail:      row.Detail,
		Action:      row.Action,
		AssessedAt:  row.AssessedAt,
	}
	if row.DueDate.Valid {
		date := row.DueDate.Time.Format("2006-01-02")
		item.DueDate = &date
	}
	return item
}

func validCategory(value string) bool {
	switch value {
	case string(dbgen.ComplianceCategoryFinancial),
		string(dbgen.ComplianceCategoryGovernance),
		string(dbgen.ComplianceCategoryAdministrative),
		string(dbgen.ComplianceCategoryInsurance):
		return true
	}
	return false
}

func validStatus(value string) bool {
	switch value {
	case string(dbgen.ComplianceStatusCompliant),
		string(dbgen.ComplianceStatusAtRisk),
		string(dbgen.ComplianceStatusNonCompliant):
		return true
	}
	return false
}
