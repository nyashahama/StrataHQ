package compliance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
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
