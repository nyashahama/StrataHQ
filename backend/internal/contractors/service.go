package contractors

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

type ContractorInfo struct {
	Phone             *string   `json:"phone"`
	Email             *string   `json:"email"`
	Notes             *string   `json:"notes,omitempty"`
	ID                string    `json:"id"`
	OrgID             string    `json:"org_id"`
	Name              string    `json:"name"`
	Trade             string    `json:"trade"`
	Suburb            string    `json:"suburb"`
	City              string    `json:"city"`
	Province          string    `json:"province"`
	SchemeIDs         []string  `json:"scheme_ids"`
	AverageRating     float64   `json:"average_rating"`
	ReviewCount       int32     `json:"review_count"`
	CompletedJobCount int32     `json:"completed_job_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	PublicProfile     bool      `json:"public_profile"`
	Vetted            bool      `json:"vetted"`
	Active            bool      `json:"active"`
	Preferred         bool      `json:"preferred"`
}

type ContractorReviewInfo struct {
	Comment              *string   `json:"comment"`
	ID                   string    `json:"id"`
	ContractorID         string    `json:"contractor_id"`
	SchemeID             string    `json:"scheme_id"`
	MaintenanceRequestID string    `json:"maintenance_request_id"`
	CreatedByUserID      string    `json:"created_by_user_id"`
	Rating               int32     `json:"rating"`
	CreatedAt            time.Time `json:"created_at"`
}

type UpsertContractorInput struct {
	Phone         *string
	Email         *string
	Notes         *string
	Name          string
	Trade         string
	Suburb        string
	City          string
	Province      string
	SchemeIDs     []string
	PublicProfile bool
	Vetted        bool
	Active        bool
}

type ContractorFilters struct {
	SchemeID string
	Trade    string
	Suburb   string
	Query    string
	Vetted   *bool
	Active   *bool
}

type ReviewInput struct {
	Comment              *string
	SchemeID             string
	MaintenanceRequestID string
	Rating               int32
}

func validRating(rating int32) bool {
	return rating >= 1 && rating <= 5
}

func normalizeRequired(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	return trimmed, trimmed != ""
}

func (s *Service) Create(ctx context.Context, identity auth.Identity, input UpsertContractorInput) (*ContractorInfo, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, userID, err := parseOrgUser(identity)
	if err != nil {
		return nil, err
	}
	params, schemeIDs, err := s.contractorParams(ctx, orgID, input)
	if err != nil {
		return nil, err
	}
	params.CreatedByUserID = pgtype.UUID{Bytes: userID, Valid: true}
	row, err := s.db.Q.CreateContractor(ctx, params)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := s.replaceSchemeLinks(ctx, row.ID, schemeIDs); err != nil {
		return nil, err
	}
	return s.mapContractor(ctx, row, auth.IsAdminRole(identity.Role))
}

func (s *Service) Update(ctx context.Context, identity auth.Identity, contractorID string, input UpsertContractorInput) (*ContractorInfo, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, _, err := parseOrgUser(identity)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(contractorID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	params, schemeIDs, err := s.contractorParams(ctx, orgID, input)
	if err != nil {
		return nil, err
	}
	row, err := s.db.Q.UpdateContractor(ctx, dbgen.UpdateContractorParams{
		ID: id, OrgID: orgID, Name: params.Name, Trade: params.Trade,
		Phone: params.Phone, Email: params.Email, Suburb: params.Suburb,
		City: params.City, Province: params.Province,
		PublicProfile: params.PublicProfile, Vetted: params.Vetted,
		Active: params.Active, Notes: params.Notes,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := s.replaceSchemeLinks(ctx, row.ID, schemeIDs); err != nil {
		return nil, err
	}
	return s.mapContractor(ctx, row, true)
}

func (s *Service) List(ctx context.Context, identity auth.Identity, filters ContractorFilters) ([]ContractorInfo, error) {
	if auth.IsResidentRole(identity.Role) {
		return nil, ErrForbidden
	}
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if !auth.IsAdminRole(identity.Role) {
		if filters.SchemeID == "" {
			return nil, ErrForbidden
		}
		if _, accessErr := s.ensureSchemeAccess(ctx, identity, filters.SchemeID); accessErr != nil {
			return nil, accessErr
		}
	}
	params := dbgen.ListContractorsByOrgParams{OrgID: orgID}
	params.SchemeID = uuidFilter(filters.SchemeID)
	params.Trade = tradeFilter(filters.Trade)
	params.Suburb = textFilter(filters.Suburb)
	params.Query = textFilter(filters.Query)
	if filters.Vetted != nil {
		params.Vetted = pgtype.Bool{Bool: *filters.Vetted, Valid: true}
	}
	if filters.Active != nil {
		params.Active = pgtype.Bool{Bool: *filters.Active, Valid: true}
	}
	rows, err := s.db.Q.ListContractorsByOrg(ctx, params)
	if err != nil {
		return nil, err
	}
	result := make([]ContractorInfo, 0, len(rows))
	for _, row := range rows {
		mapped, mapErr := s.mapContractorListRow(ctx, row, auth.IsAdminRole(identity.Role))
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (s *Service) Marketplace(ctx context.Context, identity auth.Identity, schemeID string, filters ContractorFilters) ([]ContractorInfo, error) {
	if auth.IsResidentRole(identity.Role) {
		return nil, ErrForbidden
	}
	sid, err := s.ensureSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Q.SearchContractorMarketplace(ctx, dbgen.SearchContractorMarketplaceParams{
		SchemeID: sid, Trade: tradeFilter(filters.Trade), Suburb: textFilter(filters.Suburb),
	})
	if err != nil {
		return nil, err
	}
	result := make([]ContractorInfo, 0, len(rows))
	for _, row := range rows {
		mapped, mapErr := s.mapMarketplaceRow(ctx, row, auth.IsAdminRole(identity.Role) && row.OrgID.String() == identity.OrgID)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (s *Service) CreateReview(ctx context.Context, identity auth.Identity, contractorID string, input ReviewInput) (*ContractorReviewInfo, error) {
	if auth.IsResidentRole(identity.Role) {
		return nil, ErrForbidden
	}
	if !validRating(input.Rating) {
		return nil, ErrInvalidInput
	}
	cid, parseCidErr := uuid.Parse(contractorID)
	if parseCidErr != nil {
		return nil, ErrInvalidInput
	}
	sid, parseSidErr := uuid.Parse(input.SchemeID)
	if parseSidErr != nil {
		return nil, ErrInvalidInput
	}
	if _, accessErr := s.ensureSchemeAccess(ctx, identity, input.SchemeID); accessErr != nil {
		return nil, accessErr
	}
	rid, parseRidErr := uuid.Parse(input.MaintenanceRequestID)
	if parseRidErr != nil {
		return nil, ErrInvalidInput
	}
	uid, parseUidErr := uuid.Parse(identity.UserID)
	if parseUidErr != nil {
		return nil, ErrInvalidInput
	}
	request, err := s.db.Q.GetMaintenanceRequest(ctx, rid)
	if err != nil {
		return nil, mapDBError(err)
	}
	if request.SchemeID != sid || !request.ContractorID.Valid || uuid.UUID(request.ContractorID.Bytes) != cid || request.Status != dbgen.MaintenanceStatusResolved {
		return nil, ErrForbidden
	}
	row, err := s.db.Q.CreateContractorReview(ctx, dbgen.CreateContractorReviewParams{
		ContractorID: cid, SchemeID: sid, MaintenanceRequestID: rid,
		Rating: input.Rating, Comment: optionalText(input.Comment), CreatedByUserID: uid,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	return mapReview(row), nil
}

func (s *Service) ensureSchemeAccess(ctx context.Context, identity auth.Identity, schemeID string) (uuid.UUID, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return uuid.Nil, ErrInvalidInput
	}
	scheme, err := s.db.Q.GetScheme(ctx, sid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return uuid.Nil, ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return uuid.Nil, ErrForbidden
		}
		return sid, nil
	}
	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return uuid.Nil, ErrInvalidInput
	}
	if _, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{UserID: userID, SchemeID: sid}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrForbidden
		}
		return uuid.Nil, err
	}
	return sid, nil
}

func parseOrgUser(identity auth.Identity) (uuid.UUID, uuid.UUID, error) {
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	return orgID, userID, nil
}

func (s *Service) contractorParams(ctx context.Context, orgID uuid.UUID, input UpsertContractorInput) (dbgen.CreateContractorParams, []uuid.UUID, error) {
	name, ok := normalizeRequired(input.Name)
	if !ok {
		return dbgen.CreateContractorParams{}, nil, ErrInvalidInput
	}
	suburb, ok := normalizeRequired(input.Suburb)
	if !ok || !validTrade(input.Trade) {
		return dbgen.CreateContractorParams{}, nil, ErrInvalidInput
	}
	city := strings.TrimSpace(input.City)
	if city == "" {
		city = "Cape Town"
	}
	province := strings.TrimSpace(input.Province)
	if province == "" {
		province = "Western Cape"
	}
	schemeIDs, err := parseUUIDs(input.SchemeIDs)
	if err != nil {
		return dbgen.CreateContractorParams{}, nil, ErrInvalidInput
	}
	if len(schemeIDs) > 0 {
		count, countErr := s.db.Q.CountSchemesByOrgForContractorLinks(ctx, dbgen.CountSchemesByOrgForContractorLinksParams{
			OrgID: orgID, SchemeIds: schemeIDs,
		})
		if countErr != nil {
			return dbgen.CreateContractorParams{}, nil, countErr
		}
		if count != int32(len(schemeIDs)) {
			return dbgen.CreateContractorParams{}, nil, ErrForbidden
		}
	}
	email, err := optionalEmailText(input.Email)
	if err != nil {
		return dbgen.CreateContractorParams{}, nil, ErrInvalidInput
	}
	return dbgen.CreateContractorParams{
		OrgID: orgID, Name: name, Trade: dbgen.MaintenanceCategory(input.Trade),
		Phone: optionalText(input.Phone), Email: email,
		Suburb: suburb, City: city, Province: province,
		PublicProfile: input.PublicProfile, Vetted: input.Vetted, Active: input.Active,
		Notes: optionalText(input.Notes),
	}, schemeIDs, nil
}

func validTrade(value string) bool {
	switch dbgen.MaintenanceCategory(value) {
	case dbgen.MaintenanceCategoryPlumbing, dbgen.MaintenanceCategoryElectrical, dbgen.MaintenanceCategoryStructural, dbgen.MaintenanceCategoryGarden, dbgen.MaintenanceCategoryPool, dbgen.MaintenanceCategoryOther:
		return true
	default:
		return false
	}
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) replaceSchemeLinks(ctx context.Context, contractorID uuid.UUID, schemeIDs []uuid.UUID) error {
	if err := s.db.Q.DeleteContractorSchemeLinks(ctx, contractorID); err != nil {
		return err
	}
	for _, schemeID := range schemeIDs {
		if err := s.db.Q.UpsertSchemeContractor(ctx, dbgen.UpsertSchemeContractorParams{
			SchemeID: schemeID, ContractorID: contractorID, Preferred: false,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) mapContractor(ctx context.Context, row dbgen.Contractor, includePrivate bool) (*ContractorInfo, error) {
	schemeRows, err := s.db.Q.ListContractorSchemeIDs(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	schemeIDs := make([]string, 0, len(schemeRows))
	for _, schemeID := range schemeRows {
		schemeIDs = append(schemeIDs, schemeID.String())
	}
	info := &ContractorInfo{
		ID: row.ID.String(), OrgID: row.OrgID.String(), Name: row.Name,
		Trade: string(row.Trade), Suburb: row.Suburb, City: row.City, Province: row.Province,
		PublicProfile: row.PublicProfile, Vetted: row.Vetted, Active: row.Active,
		SchemeIDs: schemeIDs, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if includePrivate {
		info.Phone = textPtr(row.Phone)
		info.Email = textPtr(row.Email)
		info.Notes = textPtr(row.Notes)
	}
	return info, nil
}

func (s *Service) mapContractorListRow(ctx context.Context, row dbgen.ListContractorsByOrgRow, includePrivate bool) (ContractorInfo, error) {
	base, err := s.mapContractor(ctx, dbgen.Contractor{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Trade: row.Trade, Phone: row.Phone,
		Email: row.Email, Suburb: row.Suburb, City: row.City, Province: row.Province,
		PublicProfile: row.PublicProfile, Vetted: row.Vetted, Active: row.Active,
		Notes: row.Notes, CreatedByUserID: row.CreatedByUserID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, includePrivate)
	if err != nil {
		return ContractorInfo{}, err
	}
	base.AverageRating = row.AverageRating
	base.ReviewCount = row.ReviewCount
	base.CompletedJobCount = row.CompletedJobCount
	base.Preferred = row.Preferred
	return *base, nil
}

func (s *Service) mapMarketplaceRow(ctx context.Context, row dbgen.SearchContractorMarketplaceRow, includePrivate bool) (ContractorInfo, error) {
	base, err := s.mapContractor(ctx, dbgen.Contractor{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Trade: row.Trade, Phone: row.Phone,
		Email: row.Email, Suburb: row.Suburb, City: row.City, Province: row.Province,
		PublicProfile: row.PublicProfile, Vetted: row.Vetted, Active: row.Active,
		Notes: row.Notes, CreatedByUserID: row.CreatedByUserID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, includePrivate)
	if err != nil {
		return ContractorInfo{}, err
	}
	base.AverageRating = row.AverageRating
	base.ReviewCount = row.ReviewCount
	base.CompletedJobCount = row.CompletedJobCount
	base.Preferred = row.Preferred
	return *base, nil
}

func mapReview(row dbgen.ContractorReview) *ContractorReviewInfo {
	return &ContractorReviewInfo{
		ID: row.ID.String(), ContractorID: row.ContractorID.String(),
		SchemeID: row.SchemeID.String(), MaintenanceRequestID: row.MaintenanceRequestID.String(),
		Rating: row.Rating, Comment: textPtr(row.Comment),
		CreatedByUserID: row.CreatedByUserID.String(), CreatedAt: row.CreatedAt,
	}
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func optionalEmailText(value *string) (pgtype.Text, error) {
	if value == nil {
		return pgtype.Text{}, nil
	}
	email := strings.TrimSpace(*value)
	if email == "" {
		return pgtype.Text{}, nil
	}
	if len(email) > 254 {
		return pgtype.Text{}, errors.New("email is too long")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return pgtype.Text{}, err
	}
	if addr.Name != "" || addr.Address != email {
		return pgtype.Text{}, errors.New("invalid email format")
	}
	local := strings.SplitN(addr.Address, "@", 2)[0]
	if local == "" || !strings.Contains(addr.Address, "@") {
		return pgtype.Text{}, errors.New("invalid email format")
	}
	return pgtype.Text{String: strings.ToLower(addr.Address), Valid: true}, nil
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func uuidFilter(raw string) pgtype.UUID {
	if raw == "" {
		return pgtype.UUID{}
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func tradeFilter(raw string) dbgen.NullMaintenanceCategory {
	if raw == "" || !validTrade(raw) {
		return dbgen.NullMaintenanceCategory{}
	}
	return dbgen.NullMaintenanceCategory{MaintenanceCategory: dbgen.MaintenanceCategory(raw), Valid: true}
}

func textFilter(raw string) pgtype.Text {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func mapDBError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}
