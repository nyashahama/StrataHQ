package whatsapp

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

const schemeWhatsAppNumber = "+27 82 787 2848"

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type MessageInfo struct {
	ID     string    `json:"id"`
	From   string    `json:"from"`
	Text   string    `json:"text"`
	SentAt time.Time `json:"sent_at"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type ThreadInfo struct {
	PhoneNumber    *string       `json:"phone_number"`
	Messages       []MessageInfo `json:"messages"`
	ID             string        `json:"id"`
	UnitID         string        `json:"unit_id"`
	UnitIdentifier string        `json:"unit_identifier"`
	OwnerName      string        `json:"owner_name"`
	Connected      bool          `json:"connected"`
	LastActive     time.Time     `json:"last_active"`
	Unread         int           `json:"unread"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type BroadcastInfo struct {
	SentByName     *string   `json:"sent_by_name"`
	ID             string    `json:"id"`
	SchemeID       string    `json:"scheme_id"`
	Message        string    `json:"message"`
	Type           string    `json:"type"`
	SentAt         time.Time `json:"sent_at"`
	RecipientCount int       `json:"recipient_count"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type DashboardResponse struct {
	ResidentThread *ThreadInfo     `json:"resident_thread"`
	Threads        []ThreadInfo    `json:"threads"`
	Broadcasts     []BroadcastInfo `json:"broadcasts"`
	Role           string          `json:"role"`
	PhoneNumber    string          `json:"phone_number"`
	TotalResidents int             `json:"total_residents"`
	ConnectedCount int             `json:"connected_count"`
	UnreadCount    int             `json:"unread_count"`
}

type CreateBroadcastInput struct {
	Message string
	Type    string
}

type accessInfo struct {
	scheme       dbgen.Scheme
	role         string
	userID       string
	memberUnitID *uuid.UUID
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
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
		Threads:     []ThreadInfo{},
		Broadcasts:  []BroadcastInfo{},
		Role:        access.role,
		PhoneNumber: schemeWhatsAppNumber,
	}

	for _, row := range threadRows {
		messages, msgErr := s.db.Q.ListWhatsAppMessagesByThread(ctx, row.ID)
		if msgErr != nil {
			return nil, msgErr
		}

		thread := mapThread(row, messages)
		response.TotalResidents++
		if row.Connected {
			response.ConnectedCount++
		}
		response.UnreadCount += int(row.UnreadCount)

		if auth.IsResidentRole(access.role) {
			if sameUnit(row.UnitID, access.memberUnitID) {
				copy := thread
				response.ResidentThread = &copy
			}
			continue
		}

		response.Threads = append(response.Threads, thread)
	}

	broadcastRows, err := s.db.Q.ListWhatsAppBroadcastsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range broadcastRows {
		response.Broadcasts = append(response.Broadcasts, BroadcastInfo{
			SentByName:     textPointer(row.SentByName),
			ID:             row.ID.String(),
			SchemeID:       row.SchemeID.String(),
			Message:        row.Message,
			Type:           string(row.Type),
			SentAt:         row.SentAt,
			RecipientCount: int(row.RecipientCount),
		})
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

	threadRows, err := s.db.Q.ListWhatsAppThreadsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	now := created.SentAt
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

func mapThread(row dbgen.ListWhatsAppThreadsDetailedBySchemeRow, messages []dbgen.WhatsappMessage) ThreadInfo {
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
		thread.Messages = append(thread.Messages, MessageInfo{
			ID:     message.ID.String(),
			From:   string(message.Sender),
			Text:   message.Body,
			SentAt: message.CreatedAt,
		})
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
