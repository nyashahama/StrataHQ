package documents

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
type DocumentInfo struct {
	UploadedByName *string   `json:"uploaded_by_name"`
	ID             string    `json:"id"`
	SchemeID       string    `json:"scheme_id"`
	Name           string    `json:"name"`
	StorageKey     string    `json:"storage_key"`
	FileType       string    `json:"file_type"`
	Category       string    `json:"category"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"uploaded_at"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type DashboardResponse struct {
	Documents []DocumentInfo `json:"documents"`
	Role      string         `json:"role"`
	Total     int            `json:"total"`
}

type CreateDocumentInput struct {
	Name       string
	StorageKey string
	FileType   string
	Category   string
	SizeBytes  int64
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

func (s *Service) List(ctx context.Context, identity auth.Identity, schemeID, category string) (*DashboardResponse, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	filter := strings.TrimSpace(category)
	if filter != "" && !validCategory(filter) {
		return nil, ErrInvalidInput
	}

	var documents []DocumentInfo
	if filter == "" {
		rows, err := s.db.Q.ListSchemeDocumentsDetailed(ctx, access.scheme.ID)
		if err != nil {
			return nil, err
		}
		documents = make([]DocumentInfo, 0, len(rows))
		for _, row := range rows {
			documents = append(documents, mapDocumentRow(row))
		}
	} else {
		rows, err := s.db.Q.ListSchemeDocumentsDetailedByCategory(ctx, dbgen.ListSchemeDocumentsDetailedByCategoryParams{
			SchemeID: access.scheme.ID,
			Category: dbgen.DocumentCategory(filter),
		})
		if err != nil {
			return nil, err
		}
		documents = make([]DocumentInfo, 0, len(rows))
		for _, row := range rows {
			documents = append(documents, mapDocumentCategoryRow(row))
		}
	}

	return &DashboardResponse{
		Documents: documents,
		Role:      access.role,
		Total:     len(documents),
	}, nil
}

func (s *Service) Create(ctx context.Context, identity auth.Identity, schemeID string, input CreateDocumentInput) (*DocumentInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(access.role) {
		return nil, ErrForbidden
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.StorageKey) == "" || input.SizeBytes < 0 || !validFileType(input.FileType) || !validCategory(input.Category) {
		return nil, ErrInvalidInput
	}

	var uploadedBy pgtype.UUID
	if access.userID != "" {
		parsed, parseErr := uuid.Parse(access.userID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		uploadedBy = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	created, err := s.db.Q.CreateSchemeDocument(ctx, dbgen.CreateSchemeDocumentParams{
		SchemeID:         access.scheme.ID,
		Name:             strings.TrimSpace(input.Name),
		StorageKey:       strings.TrimSpace(input.StorageKey),
		FileType:         dbgen.DocumentFileType(input.FileType),
		Category:         dbgen.DocumentCategory(input.Category),
		SizeBytes:        input.SizeBytes,
		UploadedByUserID: uploadedBy,
	})
	if err != nil {
		return nil, err
	}

	var uploadedByName *string
	if uploadedBy.Valid {
		user, userErr := s.db.Q.GetUserByID(ctx, uuid.UUID(uploadedBy.Bytes))
		if userErr == nil {
			uploadedByName = &user.FullName
		}
	}

	return &DocumentInfo{
		UploadedByName: uploadedByName,
		ID:             created.ID.String(),
		SchemeID:       created.SchemeID.String(),
		Name:           created.Name,
		StorageKey:     created.StorageKey,
		FileType:       string(created.FileType),
		Category:       string(created.Category),
		SizeBytes:      created.SizeBytes,
		CreatedAt:      created.CreatedAt,
	}, nil
}

func (s *Service) Delete(ctx context.Context, identity auth.Identity, schemeID, documentID string) error {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return err
	}
	if !auth.IsAdminRole(access.role) {
		return ErrForbidden
	}

	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return ErrInvalidInput
	}
	document, err := s.db.Q.GetSchemeDocument(ctx, docUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if document.SchemeID != access.scheme.ID {
		return ErrForbidden
	}

	return s.db.Q.DeleteSchemeDocument(ctx, document.ID)
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

func mapDocumentRow(row dbgen.ListSchemeDocumentsDetailedRow) DocumentInfo {
	return DocumentInfo{
		UploadedByName: textPointer(row.UploadedByName),
		ID:             row.ID.String(),
		SchemeID:       row.SchemeID.String(),
		Name:           row.Name,
		StorageKey:     row.StorageKey,
		FileType:       string(row.FileType),
		Category:       string(row.Category),
		SizeBytes:      row.SizeBytes,
		CreatedAt:      row.CreatedAt,
	}
}

func mapDocumentCategoryRow(row dbgen.ListSchemeDocumentsDetailedByCategoryRow) DocumentInfo {
	return DocumentInfo{
		UploadedByName: textPointer(row.UploadedByName),
		ID:             row.ID.String(),
		SchemeID:       row.SchemeID.String(),
		Name:           row.Name,
		StorageKey:     row.StorageKey,
		FileType:       string(row.FileType),
		Category:       string(row.Category),
		SizeBytes:      row.SizeBytes,
		CreatedAt:      row.CreatedAt,
	}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func validCategory(value string) bool {
	switch value {
	case string(dbgen.DocumentCategoryRules),
		string(dbgen.DocumentCategoryMinutes),
		string(dbgen.DocumentCategoryInsurance),
		string(dbgen.DocumentCategoryFinancial),
		string(dbgen.DocumentCategoryOther):
		return true
	default:
		return false
	}
}

func validFileType(value string) bool {
	switch value {
	case string(dbgen.DocumentFileTypePdf),
		string(dbgen.DocumentFileTypeDocx),
		string(dbgen.DocumentFileTypeXlsx),
		string(dbgen.DocumentFileTypeJpg),
		string(dbgen.DocumentFileTypePng):
		return true
	default:
		return false
	}
}
