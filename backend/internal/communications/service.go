package communications

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

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type NoticeInfo struct {
	SentByName *string   `json:"sent_by_name"`
	ID         string    `json:"id"`
	SchemeID   string    `json:"scheme_id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Type       string    `json:"type"`
	SentAt     time.Time `json:"sent_at"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type DashboardResponse struct {
	Notices []NoticeInfo `json:"notices"`
	Role    string       `json:"role"`
	Total   int          `json:"total"`
}

type CreateNoticeInput struct {
	Title string
	Body  string
	Type  string
}

type accessInfo struct {
	scheme dbgen.Scheme
	role   string
	userID string
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, identity auth.Identity, schemeID string, typeFilter string) (*DashboardResponse, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	filter := strings.TrimSpace(typeFilter)
	if filter != "" && !validNoticeType(filter) {
		return nil, ErrInvalidInput
	}

	var notices []NoticeInfo
	if filter == "" {
		rows, err := s.db.Q.ListNoticesDetailedByScheme(ctx, access.scheme.ID)
		if err != nil {
			return nil, err
		}
		notices = make([]NoticeInfo, 0, len(rows))
		for _, row := range rows {
			notices = append(notices, NoticeInfo{
				SentByName: textPointer(row.SentByName),
				ID:         row.ID.String(),
				SchemeID:   row.SchemeID.String(),
				Title:      row.Title,
				Body:       row.Body,
				Type:       string(row.Type),
				SentAt:     row.SentAt,
			})
		}
	} else {
		rows, err := s.db.Q.ListNoticesDetailedBySchemeAndType(ctx, dbgen.ListNoticesDetailedBySchemeAndTypeParams{
			SchemeID: access.scheme.ID,
			Type:     dbgen.NoticeType(filter),
		})
		if err != nil {
			return nil, err
		}
		notices = make([]NoticeInfo, 0, len(rows))
		for _, row := range rows {
			notices = append(notices, NoticeInfo{
				SentByName: textPointer(row.SentByName),
				ID:         row.ID.String(),
				SchemeID:   row.SchemeID.String(),
				Title:      row.Title,
				Body:       row.Body,
				Type:       string(row.Type),
				SentAt:     row.SentAt,
			})
		}
	}

	return &DashboardResponse{
		Notices: notices,
		Role:    access.role,
		Total:   len(notices),
	}, nil
}

func (s *Service) Create(ctx context.Context, identity auth.Identity, schemeID string, input CreateNoticeInput) (*NoticeInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}

	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Body) == "" || !validNoticeType(input.Type) {
		return nil, ErrInvalidInput
	}

	var sentByUserID pgtype.UUID
	if access.userID != "" {
		parsed, parseErr := uuid.Parse(access.userID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		sentByUserID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	created, err := s.db.Q.CreateNotice(ctx, dbgen.CreateNoticeParams{
		SchemeID:     access.scheme.ID,
		Title:        strings.TrimSpace(input.Title),
		Body:         strings.TrimSpace(input.Body),
		Type:         dbgen.NoticeType(input.Type),
		SentByUserID: sentByUserID,
	})
	if err != nil {
		return nil, err
	}

	var senderName *string
	if sentByUserID.Valid {
		user, userErr := s.db.Q.GetUserByID(ctx, uuid.UUID(sentByUserID.Bytes))
		if userErr == nil {
			senderName = &user.FullName
		}
	}

	return &NoticeInfo{
		SentByName: senderName,
		ID:         created.ID.String(),
		SchemeID:   created.SchemeID.String(),
		Title:      created.Title,
		Body:       created.Body,
		Type:       string(created.Type),
		SentAt:     created.SentAt,
	}, nil
}

func (s *Service) resolveAccess(ctx context.Context, identity auth.Identity, schemeID string) (*accessInfo, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return nil, ErrForbidden
		}
		return &accessInfo{scheme: scheme, role: string(auth.RoleAdmin), userID: identity.UserID}, nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return nil, ErrInvalidInput
	}
	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	return &accessInfo{scheme: scheme, role: membership.Role, userID: identity.UserID}, nil
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func validNoticeType(value string) bool {
	switch value {
	case string(dbgen.NoticeTypeGeneral),
		string(dbgen.NoticeTypeUrgent),
		string(dbgen.NoticeTypeAgm),
		string(dbgen.NoticeTypeLevy):
		return true
	default:
		return false
	}
}
