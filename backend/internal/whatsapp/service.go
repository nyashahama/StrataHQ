package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
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

const schemeWhatsAppNumber = "+27 69 785 2182"

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type MessageInfo struct {
	SentAt               time.Time   `json:"sent_at"`
	ID                   string      `json:"id"`
	From                 string      `json:"from"`
	Text                 string      `json:"text"`
	MaintenanceRequestID *string     `json:"maintenance_request_id"`
	Media                []MediaInfo `json:"media"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type ThreadInfo struct {
	LastActive     time.Time     `json:"last_active"`
	PhoneNumber    *string       `json:"phone_number"`
	ID             string        `json:"id"`
	UnitID         string        `json:"unit_id"`
	UnitIdentifier string        `json:"unit_identifier"`
	OwnerName      string        `json:"owner_name"`
	Messages       []MessageInfo `json:"messages"`
	Unread         int           `json:"unread"`
	Connected      bool          `json:"connected"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type BroadcastInfo struct {
	SentAt         time.Time `json:"sent_at"`
	SentByName     *string   `json:"sent_by_name"`
	ID             string    `json:"id"`
	SchemeID       string    `json:"scheme_id"`
	Message        string    `json:"message"`
	Type           string    `json:"type"`
	RecipientCount int       `json:"recipient_count"`
	// DeliveredRecipientCount is the number of connected threads that were sent successfully.
	DeliveredRecipientCount int `json:"delivered_recipient_count"`
	// FailedRecipientCount is the number of connected threads that did not deliver successfully.
	FailedRecipientCount int `json:"failed_recipient_count"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type DashboardResponse struct {
	ResidentThread     *ThreadInfo             `json:"resident_thread"`
	Role               string                  `json:"role"`
	PhoneNumber        string                  `json:"phone_number"`
	Threads            []ThreadInfo            `json:"threads"`
	Broadcasts         []BroadcastInfo         `json:"broadcasts"`
	TotalResidents     int                     `json:"total_residents"`
	ConnectedCount     int                     `json:"connected_count"`
	UnreadCount        int                     `json:"unread_count"`
	MaintenanceIntakes []MaintenanceIntakeInfo `json:"maintenance_intakes"`
}

type CreateBroadcastInput struct {
	Message string
	Type    string
}

type MediaInfo struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

type MaintenanceIntakeInfo struct {
	CreatedAt            time.Time `json:"created_at"`
	MaintenanceRequestID *string   `json:"maintenance_request_id"`
	ID                   string    `json:"id"`
	SchemeID             string    `json:"scheme_id"`
	ThreadID             string    `json:"thread_id"`
	MessageID            string    `json:"message_id"`
	UnitID               string    `json:"unit_id"`
	UnitIdentifier       string    `json:"unit_identifier"`
	OwnerName            string    `json:"owner_name"`
	Status               string    `json:"status"`
	Category             string    `json:"category"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	MediaCount           int       `json:"media_count"`
}

type CreateMaintenanceFromMessageInput struct {
	Title       string
	Description string
	Category    string
}

type maintenanceIntent struct {
	IsMaintenance bool
	Category      string
}

type intakeText struct {
	Title       string
	Description string
}

type accessInfo struct {
	scheme       dbgen.Scheme
	memberUnitID *uuid.UUID
	role         string
	userID       string
}

type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	db      *database.Pool
	sender  MessageSender
	logger  *slog.Logger
	auditor resourceAuditor
}

func NewService(db *database.Pool, sender MessageSender, logger *slog.Logger) *Service {
	return NewServiceWithAudit(db, sender, logger, nil)
}

func NewServiceWithAudit(db *database.Pool, sender MessageSender, logger *slog.Logger, auditor resourceAuditor) *Service {
	return &Service{db: db, sender: sender, logger: logger, auditor: auditor}
}

func (s *Service) Dashboard(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	threadRows, err := s.db.Q.ListWhatsAppThreadsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	response := &DashboardResponse{
		Threads:            []ThreadInfo{},
		Broadcasts:         []BroadcastInfo{},
		MaintenanceIntakes: []MaintenanceIntakeInfo{},
		Role:               access.role,
		PhoneNumber:        schemeWhatsAppNumber,
	}

	for _, row := range threadRows {
		messages, msgErr := s.db.Q.ListWhatsAppMessagesByThread(ctx, row.ID)
		if msgErr != nil {
			return nil, msgErr
		}

		mediaRows, mediaErr := s.db.Q.ListWhatsAppMessageMediaByThread(ctx, row.ID)
		if mediaErr != nil {
			return nil, mediaErr
		}

		thread := mapThread(row, messages, mediaRows)

		if auth.IsResidentRole(access.role) {
			if sameUnit(row.UnitID, access.memberUnitID) {
				copy := thread
				response.ResidentThread = &copy
				response.TotalResidents = 1
				if copy.Connected {
					response.ConnectedCount = 1
				}
				response.UnreadCount = copy.Unread
			}
			continue
		}

		response.TotalResidents++
		if row.Connected {
			response.ConnectedCount++
		}
		response.UnreadCount += int(row.UnreadCount)
		response.Threads = append(response.Threads, thread)
	}

	if !auth.IsResidentRole(access.role) {
		intakes, intakeErr := s.db.Q.ListWhatsAppMaintenanceIntakesByScheme(ctx, access.scheme.ID)
		if intakeErr != nil {
			return nil, intakeErr
		}
		for _, intake := range intakes {
			response.MaintenanceIntakes = append(response.MaintenanceIntakes, mapMaintenanceIntake(intake))
		}
	}

	broadcastRows, err := s.db.Q.ListWhatsAppBroadcastsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range broadcastRows {
		broadcast := BroadcastInfo{
			SentByName: textPointer(row.SentByName),
			ID:         row.ID.String(),
			SchemeID:   row.SchemeID.String(),
			Message:    row.Message,
			Type:       string(row.Type),
			SentAt:     row.SentAt,
		}
		if !auth.IsResidentRole(access.role) {
			broadcast.RecipientCount = int(row.RecipientCount)
		}
		response.Broadcasts = append(response.Broadcasts, broadcast)
	}

	return response, nil
}

func (s *Service) CreateBroadcast(ctx context.Context, identity auth.Identity, schemeID string, input CreateBroadcastInput) (*BroadcastInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}

	message := strings.TrimSpace(input.Message)
	if message == "" || !validBroadcastType(input.Type) {
		return nil, ErrInvalidInput
	}

	var sentByUserID pgtype.UUID
	if access.userID != "" {
		parsedUserID, parseErr := uuid.Parse(access.userID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		sentByUserID = pgtype.UUID{Bytes: parsedUserID, Valid: true}
	}

	connectedCount, err := s.db.Q.CountConnectedWhatsAppThreadsByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	created, err := s.db.Q.CreateWhatsAppBroadcast(ctx, dbgen.CreateWhatsAppBroadcastParams{
		SchemeID:       access.scheme.ID,
		SentByUserID:   sentByUserID,
		Type:           dbgen.WhatsappBroadcastType(input.Type),
		Message:        message,
		RecipientCount: int32(connectedCount),
	})
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, whatsAppBroadcastCreatedAuditEvent(whatsAppAuditInput{
			SchemeID:       access.scheme.ID.String(),
			OrgID:          access.scheme.OrgID.String(),
			ActorUserID:    access.userID,
			ActorRole:      access.role,
			BroadcastID:    created.ID.String(),
			Message:        created.Message,
			Type:           string(created.Type),
			RecipientCount: int(created.RecipientCount),
			SentAt:         created.SentAt,
		}))
	}

	threadRows, err := s.db.Q.ListWhatsAppThreadsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	now := created.SentAt
	sendErrors := 0
	deliveredRecipientCount := 0
	for _, row := range threadRows {
		if !row.Connected {
			continue
		}

		if _, err := s.db.Q.CreateWhatsAppMessage(ctx, dbgen.CreateWhatsAppMessageParams{
			ThreadID:             row.ID,
			Sender:               dbgen.WhatsappMessageSenderBot,
			Body:                 message,
			MaintenanceRequestID: pgtype.UUID{},
			NoticeID:             pgtype.UUID{},
		}); err != nil {
			return nil, err
		}

		if err := s.db.Q.TouchWhatsAppThread(ctx, dbgen.TouchWhatsAppThreadParams{
			ID:           row.ID,
			UnreadCount:  row.UnreadCount,
			LastActiveAt: now,
		}); err != nil {
			return nil, err
		}

		if row.PhoneNumber.Valid && row.PhoneNumber.String != "" {
			if err := s.sender.SendWhatsAppMessage(row.PhoneNumber.String, message); err != nil {
				s.logger.Error("failed to send WhatsApp broadcast", "phone", row.PhoneNumber.String, "error", err)
				sendErrors++
			} else {
				deliveredRecipientCount++
			}
		} else {
			sendErrors++
		}
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, whatsAppBroadcastSentAuditEvent(whatsAppAuditInput{
			SchemeID:       access.scheme.ID.String(),
			OrgID:          access.scheme.OrgID.String(),
			ActorUserID:    access.userID,
			ActorRole:      access.role,
			BroadcastID:    created.ID.String(),
			Message:        created.Message,
			Type:           string(created.Type),
			RecipientCount: int(created.RecipientCount),
			SentAt:         created.SentAt,
		}, sendErrors))
	}

	var senderName *string
	if sentByUserID.Valid {
		user, userErr := s.db.Q.GetUserByID(ctx, uuid.UUID(sentByUserID.Bytes))
		if userErr == nil {
			senderName = &user.FullName
		}
	}

	return &BroadcastInfo{
		SentByName:     senderName,
		ID:             created.ID.String(),
		SchemeID:       created.SchemeID.String(),
		Message:        created.Message,
		Type:           string(created.Type),
		SentAt:         created.SentAt,
		RecipientCount: int(created.RecipientCount),
		DeliveredRecipientCount: deliveredRecipientCount,
		FailedRecipientCount:    sendErrors,
	}, nil
}

func (s *Service) resolveAccess(ctx context.Context, identity auth.Identity, schemeID string) (*accessInfo, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, sid)
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
		SchemeID: sid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	access := &accessInfo{
		scheme: scheme,
		role:   membership.Role,
		userID: identity.UserID,
	}
	if membership.UnitID.Valid {
		unitID := uuid.UUID(membership.UnitID.Bytes)
		access.memberUnitID = &unitID
	}

	return access, nil
}

func mapThread(row dbgen.ListWhatsAppThreadsDetailedBySchemeRow, messages []dbgen.WhatsappMessage, mediaRows []dbgen.WhatsappMessageMedium) ThreadInfo {
	mediaByMessage := make(map[uuid.UUID][]MediaInfo)
	for _, m := range mediaRows {
		contentType := ""
		if m.ContentType.Valid {
			contentType = m.ContentType.String
		}
		mediaByMessage[m.MessageID] = append(mediaByMessage[m.MessageID], MediaInfo{
			ID:          m.ID.String(),
			URL:         m.MediaUrl,
			ContentType: contentType,
		})
	}

	thread := ThreadInfo{
		Messages:       make([]MessageInfo, 0, len(messages)),
		ID:             row.ID.String(),
		UnitID:         row.UnitID.String(),
		UnitIdentifier: row.UnitIdentifier,
		OwnerName:      row.OwnerName,
		Connected:      row.Connected,
		LastActive:     row.LastActiveAt,
		Unread:         int(row.UnreadCount),
	}
	if row.PhoneNumber.Valid {
		phone := row.PhoneNumber.String
		thread.PhoneNumber = &phone
	}
	for _, message := range messages {
		info := MessageInfo{
			ID:     message.ID.String(),
			From:   string(message.Sender),
			Text:   message.Body,
			SentAt: message.CreatedAt,
			Media:  mediaByMessage[message.ID],
		}
		if message.MaintenanceRequestID.Valid {
			id := uuid.UUID(message.MaintenanceRequestID.Bytes).String()
			info.MaintenanceRequestID = &id
		}
		thread.Messages = append(thread.Messages, info)
	}
	return thread
}

func sameUnit(unitID uuid.UUID, memberUnitID *uuid.UUID) bool {
	return memberUnitID != nil && unitID == *memberUnitID
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func validBroadcastType(value string) bool {
	switch value {
	case string(dbgen.WhatsappBroadcastTypeGeneral),
		string(dbgen.WhatsappBroadcastTypeAgm),
		string(dbgen.WhatsappBroadcastTypeLevy),
		string(dbgen.WhatsappBroadcastTypeMaintenance):
		return true
	default:
		return false
	}
}

func classifyMaintenanceIntent(body string, mediaCount int) maintenanceIntent {
	text := strings.ToLower(strings.TrimSpace(body))
	category := inferMaintenanceCategory(text)
	intentPrefixes := []string{"2", "request", "maintenance", "repair", "fix", "broken", "leak", "burst", "blocked", "no power", "lights", "plug", "gate", "roof", "crack", "pool", "garden"}
	for _, prefix := range intentPrefixes {
		if text == prefix || strings.HasPrefix(text, prefix+" ") {
			return maintenanceIntent{IsMaintenance: true, Category: category}
		}
	}
	if mediaCount > 0 && category != "other" {
		return maintenanceIntent{IsMaintenance: true, Category: category}
	}
	return maintenanceIntent{IsMaintenance: false, Category: category}
}

func shouldCreateMaintenanceCandidate(intent maintenanceIntent, mediaCount int) bool {
	return !intent.IsMaintenance && (mediaCount > 0 || intent.Category != "other")
}

func inferMaintenanceCategory(text string) string {
	switch {
	case containsAny(text, "leak", "water", "tap", "toilet", "drain", "blocked", "burst", "geyser"):
		return "plumbing"
	case containsAny(text, "power", "light", "plug", "electric", "gate", "intercom"):
		return "electrical"
	case containsAny(text, "roof", "wall", "crack", "ceiling", "door", "window"):
		return "structural"
	case containsAny(text, "garden", "tree", "grass", "irrigation"):
		return "garden"
	case containsAny(text, "pool", "pump", "chlorine"):
		return "pool"
	default:
		return "other"
	}
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func buildMaintenanceIntakeText(body string, mediaCount int) intakeText {
	clean := strings.TrimSpace(body)
	if clean == "" && mediaCount > 0 {
		clean = "Maintenance photo received via WhatsApp"
	}
	titleSource := clean
	if idx := strings.Index(titleSource, "."); idx > 0 {
		titleSource = titleSource[:idx]
	}
	if len(titleSource) > 80 {
		runes := []rune(titleSource)
		titleSource = string(runes[:80])
	}
	result := intakeText{
		Title:       "WhatsApp: " + strings.TrimSpace(titleSource),
		Description: clean,
	}
	if mediaCount > 0 {
		result.Description = strings.TrimSpace(result.Description + "\n\nMedia attached: " + strconv.Itoa(mediaCount) + " item(s)")
	}
	return result
}

type whatsAppAuditInput struct {
	SchemeID       string
	OrgID          string
	ActorUserID    string
	ActorRole      string
	BroadcastID    string
	Message        string
	Type           string
	RecipientCount int
	SentAt         time.Time
}

func whatsAppBroadcastCreatedAuditEvent(input whatsAppAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "whatsapp_broadcast",
		ResourceID:   input.BroadcastID,
		Action:       "whatsapp.broadcast_created",
		AfterState: map[string]any{
			"message":         input.Message,
			"type":            input.Type,
			"recipient_count": input.RecipientCount,
			"sent_at":         input.SentAt.Format(time.RFC3339),
		},
	}
}

func whatsAppBroadcastSentAuditEvent(input whatsAppAuditInput, sendErrors int) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "whatsapp_broadcast",
		ResourceID:   input.BroadcastID,
		Action:       "whatsapp.broadcast_sent",
		AfterState: map[string]any{
			"message":         input.Message,
			"type":            input.Type,
			"recipient_count": input.RecipientCount,
			"sent_at":         input.SentAt.Format(time.RFC3339),
		},
		Metadata: map[string]any{
			"send_errors": sendErrors,
		},
	}
}

func mapMaintenanceIntake(row dbgen.ListWhatsAppMaintenanceIntakesBySchemeRow) MaintenanceIntakeInfo {
	info := MaintenanceIntakeInfo{
		ID:             row.ID.String(),
		SchemeID:       row.SchemeID.String(),
		ThreadID:       row.ThreadID.String(),
		MessageID:      row.MessageID.String(),
		UnitID:         row.UnitID.String(),
		UnitIdentifier: row.UnitIdentifier,
		OwnerName:      row.OwnerName,
		Status:         row.Status,
		Category:       string(row.Category),
		Title:          row.Title,
		Description:    row.Description,
		MediaCount:     int(row.MediaCount),
		CreatedAt:      row.CreatedAt,
	}
	if row.MaintenanceRequestID.Valid {
		id := uuid.UUID(row.MaintenanceRequestID.Bytes).String()
		info.MaintenanceRequestID = &id
	}
	return info
}

func (s *Service) CreateMaintenanceFromMessage(ctx context.Context, identity auth.Identity, schemeID, messageID string, input CreateMaintenanceFromMessageInput) (*MaintenanceIntakeInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	msg, err := s.db.Q.GetWhatsAppMessageWithThread(ctx, mid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if msg.SchemeID != access.scheme.ID {
		return nil, ErrForbidden
	}
	intakes, intakeErr := s.db.Q.ListWhatsAppMaintenanceIntakesByScheme(ctx, access.scheme.ID)
	if intakeErr != nil {
		return nil, intakeErr
	}
	for _, row := range intakes {
		if row.MessageID != mid {
			continue
		}
		if row.Status == "dismissed" {
			return nil, ErrInvalidInput
		}
		if msg.MaintenanceRequestID.Valid {
			mapped := mapMaintenanceIntake(row)
			return &mapped, nil
		}
	}

	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	category := strings.TrimSpace(input.Category)
	if title == "" || description == "" || !validMaintenanceCategory(category) {
		return nil, ErrInvalidInput
	}
	mediaCount, err := s.db.Q.CountWhatsAppMessageMediaByMessage(ctx, mid)
	if err != nil {
		return nil, err
	}
	return s.createMaintenanceTicketForMessage(ctx, access.scheme.ID, msg.ThreadID, mid, msg.UnitID, title, description, category, int(mediaCount))
}

func (s *Service) DismissMaintenanceIntake(ctx context.Context, identity auth.Identity, schemeID, intakeID string) (*MaintenanceIntakeInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}
	id, err := uuid.Parse(intakeID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	intake, err := s.db.Q.GetWhatsAppMaintenanceIntake(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if intake.SchemeID != access.scheme.ID {
		return nil, ErrForbidden
	}
	if intake.Status != "candidate" {
		return nil, ErrInvalidInput
	}
	dismissed, err := s.db.Q.DismissWhatsAppMaintenanceIntake(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Q.ListWhatsAppMaintenanceIntakesByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.ID == dismissed.ID {
			mapped := mapMaintenanceIntake(item)
			return &mapped, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) createMaintenanceTicketForMessage(ctx context.Context, schemeID, threadID, messageID, unitID uuid.UUID, title, description, category string, mediaCount int) (*MaintenanceIntakeInfo, error) {
	unit, err := s.db.Q.GetUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	var intake dbgen.WhatsappMaintenanceIntake
	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		request, txErr := q.CreateMaintenanceRequest(ctx, dbgen.CreateMaintenanceRequestParams{
			SchemeID:        schemeID,
			UnitID:          pgtype.UUID{Bytes: unitID, Valid: true},
			Title:           title,
			Description:     description,
			Category:        dbgen.MaintenanceCategory(category),
			SlaHours:        defaultWhatsAppSLAHours(category),
			SubmittedByUnit: pgtype.Text{String: unit.Identifier, Valid: true},
		})
		if txErr != nil {
			return txErr
		}
		request, txErr = q.UpdateMaintenanceStatus(ctx, dbgen.UpdateMaintenanceStatusParams{
			ID:     request.ID,
			Status: dbgen.MaintenanceStatusPendingApproval,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = q.UpdateWhatsAppMessageMaintenanceRequest(ctx, dbgen.UpdateWhatsAppMessageMaintenanceRequestParams{
			ID:                   messageID,
			MaintenanceRequestID: pgtype.UUID{Bytes: request.ID, Valid: true},
		}); txErr != nil {
			return txErr
		}
		intake, txErr = q.CreateWhatsAppMaintenanceIntake(ctx, dbgen.CreateWhatsAppMaintenanceIntakeParams{
			SchemeID:             schemeID,
			ThreadID:             threadID,
			MessageID:            messageID,
			UnitID:               unitID,
			MaintenanceRequestID: pgtype.UUID{Bytes: request.ID, Valid: true},
			Status:               "ticket_created",
			Category:             dbgen.MaintenanceCategory(category),
			Title:                title,
			Description:          description,
			MediaCount:           int32(mediaCount),
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Q.ListWhatsAppMaintenanceIntakesByScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.ID == intake.ID {
			mapped := mapMaintenanceIntake(row)
			return &mapped, nil
		}
	}
	return nil, ErrNotFound
}

func validMaintenanceCategory(category string) bool {
	switch category {
	case string(dbgen.MaintenanceCategoryPlumbing),
		string(dbgen.MaintenanceCategoryElectrical),
		string(dbgen.MaintenanceCategoryStructural),
		string(dbgen.MaintenanceCategoryGarden),
		string(dbgen.MaintenanceCategoryPool),
		string(dbgen.MaintenanceCategoryOther):
		return true
	default:
		return false
	}
}

func defaultWhatsAppSLAHours(category string) int32 {
	switch category {
	case string(dbgen.MaintenanceCategoryElectrical),
		string(dbgen.MaintenanceCategoryPlumbing):
		return 24
	case string(dbgen.MaintenanceCategoryStructural):
		return 48
	default:
		return 72
	}
}
