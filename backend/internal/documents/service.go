package documents

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/audit"
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
	Visibility string
}

type accessInfo struct {
	scheme dbgen.Scheme
	role   string
	userID string
}

type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	db      *database.Pool
	auditor resourceAuditor
}

func NewService(db *database.Pool) *Service {
	return NewServiceWithAudit(db, nil)
}

func NewServiceWithAudit(db *database.Pool, auditor resourceAuditor) *Service {
	return &Service{db: db, auditor: auditor}
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

	visibilities := determineVisibilities(access.role)
	rows, err := s.db.Q.ListSchemeDocumentsDetailedByVisibility(ctx, dbgen.ListSchemeDocumentsDetailedByVisibilityParams{
		SchemeID: access.scheme.ID,
		Column2:  visibilities,
	})
	if err != nil {
		return nil, err
	}

	documents := make([]DocumentInfo, 0, len(rows))
	for _, row := range rows {
		if filter != "" && string(row.Category) != filter {
			continue
		}
		documents = append(documents, mapVisibilityDocumentRow(row))
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
	storageKey, err := normalizeStorageKey(input.StorageKey, input.FileType)
	if err != nil {
		return nil, err
	}

	var uploadedBy pgtype.UUID
	if access.userID != "" {
		parsed, parseErr := uuid.Parse(access.userID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		uploadedBy = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	visibility := dbgen.DocumentVisibilityAdmin
	if input.Visibility != "" {
		switch dbgen.DocumentVisibility(input.Visibility) {
		case dbgen.DocumentVisibilityAll, dbgen.DocumentVisibilityTrustee, dbgen.DocumentVisibilityAdmin:
			visibility = dbgen.DocumentVisibility(input.Visibility)
		}
	}

	created, err := s.db.Q.CreateSchemeDocument(ctx, dbgen.CreateSchemeDocumentParams{
		SchemeID:         access.scheme.ID,
		Name:             strings.TrimSpace(input.Name),
		StorageKey:       storageKey,
		FileType:         dbgen.DocumentFileType(input.FileType),
		Category:         dbgen.DocumentCategory(input.Category),
		SizeBytes:        input.SizeBytes,
		UploadedByUserID: uploadedBy,
		Visibility:       visibility,
	})
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, documentCreatedAuditEvent(documentAuditInput{
			SchemeID:    access.scheme.ID.String(),
			OrgID:       access.scheme.OrgID.String(),
			ActorUserID: access.userID,
			ActorRole:   access.role,
			DocumentID:  created.ID.String(),
			Name:        created.Name,
			Category:    string(created.Category),
			FileType:    string(created.FileType),
			SizeBytes:   created.SizeBytes,
			StorageKey:  created.StorageKey,
		}))
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
		StorageKey:     safeListedStorageKey(created.StorageKey, string(created.FileType)),
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

	if err := s.db.Q.DeleteSchemeDocument(ctx, document.ID); err != nil {
		return err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, documentDeletedAuditEvent(documentAuditInput{
			SchemeID:    access.scheme.ID.String(),
			OrgID:       access.scheme.OrgID.String(),
			ActorUserID: access.userID,
			ActorRole:   access.role,
			DocumentID:  document.ID.String(),
			Name:        document.Name,
			Category:    string(document.Category),
			FileType:    string(document.FileType),
			SizeBytes:   document.SizeBytes,
			StorageKey:  document.StorageKey,
		}))
	}

	return nil
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

var allowedDataURLPrefixes = map[string][]string{
	"pdf": {
		"data:application/pdf;base64,",
	},
	"docx": {
		"data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,",
	},
	"xlsx": {
		"data:application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;base64,",
	},
	"jpg": {
		"data:image/jpeg;base64,",
		"data:image/jpg;base64,",
	},
	"png": {
		"data:image/png;base64,",
	},
}

func normalizeStorageKey(raw, fileType string) (string, error) {
	storageKey := strings.TrimSpace(raw)
	if storageKey == "" {
		return "", ErrInvalidInput
	}

	normalized := strings.ToLower(storageKey)
	for _, prefix := range allowedDataURLPrefixes[fileType] {
		if strings.HasPrefix(normalized, prefix) {
			return storageKey, nil
		}
	}

	if !strings.HasPrefix(storageKey, "/") || strings.HasPrefix(storageKey, "//") || strings.ContainsAny(storageKey, "\r\n\t\\") {
		return "", ErrInvalidInput
	}

	parsed, err := url.Parse(storageKey)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", ErrInvalidInput
	}

	return storageKey, nil
}

func safeListedStorageKey(raw, fileType string) string {
	storageKey, err := normalizeStorageKey(raw, fileType)
	if err != nil {
		return ""
	}
	return storageKey
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

type documentAuditInput struct {
	SchemeID    string
	OrgID       string
	ActorUserID string
	ActorRole   string
	DocumentID  string
	Name        string
	Category    string
	FileType    string
	SizeBytes   int64
	StorageKey  string
}

func documentCreatedAuditEvent(input documentAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "document",
		ResourceID:   input.DocumentID,
		Action:       "document.uploaded",
		AfterState:   documentAuditState(input),
	}
}

func documentDeletedAuditEvent(input documentAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "document",
		ResourceID:   input.DocumentID,
		Action:       "document.deleted",
		BeforeState:  documentAuditState(input),
	}
}

func documentAuditState(input documentAuditInput) map[string]any {
	return map[string]any{
		"name":        input.Name,
		"category":    input.Category,
		"file_type":   input.FileType,
		"size_bytes":  input.SizeBytes,
		"storage_key": input.StorageKey,
	}
}

func determineVisibilities(role string) []dbgen.DocumentVisibility {
	switch {
	case auth.IsAdminRole(role):
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
			dbgen.DocumentVisibilityTrustee,
			dbgen.DocumentVisibilityAdmin,
		}
	case role == string(auth.RoleTrustee):
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
			dbgen.DocumentVisibilityTrustee,
		}
	default:
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
		}
	}
}

func mapVisibilityDocumentRow(row dbgen.ListSchemeDocumentsDetailedByVisibilityRow) DocumentInfo {
	return DocumentInfo{
		UploadedByName: textPointer(row.UploadedByName),
		ID:             row.ID.String(),
		SchemeID:       row.SchemeID.String(),
		Name:           row.Name,
		StorageKey:     safeListedStorageKey(row.StorageKey, string(row.FileType)),
		FileType:       string(row.FileType),
		Category:       string(row.Category),
		SizeBytes:      row.SizeBytes,
		CreatedAt:      row.CreatedAt,
	}
}
