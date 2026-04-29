package invitation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

type fakeInvitationStore struct {
	scheme                *dbgen.Scheme
	unit                  *dbgen.Unit
	invitationByID        *dbgen.Invitation
	invitationByToken     *dbgen.Invitation
	createdUser           *dbgen.User
	createdInvitation     *dbgen.CreateInvitationParams
	updatedInvitation     *dbgen.UpdateInvitationTokenParams
	createdRefreshToken   *dbgen.CreateRefreshTokenParams
	updatedInviteStatus   *dbgen.UpdateInvitationStatusParams
	schemeErr             error
	unitErr               error
	invitationByIDErr     error
	invitationByTokenErr  error
	getUserByEmailErr     error
	createUserErr         error
	createRefreshTokenErr error
}

type fakeInvitationSender struct {
	sendErr error
	calls   int
}

func (s *fakeInvitationSender) SendInvitation(context.Context, string, string, string) error {
	s.calls++
	return s.sendErr
}

func (s *fakeInvitationSender) SendPasswordReset(context.Context, string, string) error {
	return nil
}

func (s *fakeInvitationSender) SendEarlyAccessApproval(context.Context, string, string, string) error {
	return nil
}

func (s *fakeInvitationSender) SendNewEarlyAccessRequest(context.Context, string, string, string, string, int32, string, string) error {
	return nil
}

type fakeInvitationAuditor struct {
	events []audit.ResourceEvent
}

func (a *fakeInvitationAuditor) RecordResourceEvent(_ context.Context, event audit.ResourceEvent) error {
	a.events = append(a.events, event)
	return nil
}

func (f *fakeInvitationStore) CreateInvitation(_ context.Context, arg dbgen.CreateInvitationParams) (dbgen.Invitation, error) {
	f.createdInvitation = &arg
	return dbgen.Invitation{
		ID:        uuid.New(),
		OrgID:     arg.OrgID,
		SchemeID:  arg.SchemeID,
		UnitID:    arg.UnitID,
		Email:     arg.Email,
		FullName:  arg.FullName,
		Role:      arg.Role,
		Token:     arg.Token,
		Status:    "pending",
		ExpiresAt: arg.ExpiresAt,
	}, nil
}

func (f *fakeInvitationStore) ListInvitationsByOrg(context.Context, uuid.UUID) ([]dbgen.Invitation, error) {
	return nil, nil
}

func (f *fakeInvitationStore) GetInvitationByID(context.Context, uuid.UUID) (dbgen.Invitation, error) {
	if f.invitationByIDErr != nil {
		return dbgen.Invitation{}, f.invitationByIDErr
	}
	if f.invitationByID == nil {
		return dbgen.Invitation{}, pgx.ErrNoRows
	}
	return *f.invitationByID, nil
}

func (f *fakeInvitationStore) UpdateInvitationToken(_ context.Context, arg dbgen.UpdateInvitationTokenParams) (dbgen.Invitation, error) {
	f.updatedInvitation = &arg
	if f.invitationByID == nil {
		return dbgen.Invitation{}, nil
	}
	inv := *f.invitationByID
	inv.Token = arg.Token
	inv.ExpiresAt = arg.ExpiresAt
	return inv, nil
}

func (f *fakeInvitationStore) UpdateInvitationStatus(_ context.Context, arg dbgen.UpdateInvitationStatusParams) error {
	f.updatedInviteStatus = &arg
	return nil
}

func (f *fakeInvitationStore) GetInvitationByToken(context.Context, string) (dbgen.Invitation, error) {
	if f.invitationByTokenErr != nil {
		return dbgen.Invitation{}, f.invitationByTokenErr
	}
	if f.invitationByToken == nil {
		return dbgen.Invitation{}, nil
	}
	return *f.invitationByToken, nil
}

func (f *fakeInvitationStore) GetUserByEmail(context.Context, string) (dbgen.User, error) {
	if f.getUserByEmailErr != nil {
		return dbgen.User{}, f.getUserByEmailErr
	}
	return dbgen.User{}, nil
}

func (f *fakeInvitationStore) CreateRefreshToken(_ context.Context, arg dbgen.CreateRefreshTokenParams) (dbgen.RefreshToken, error) {
	f.createdRefreshToken = &arg
	if f.createRefreshTokenErr != nil {
		return dbgen.RefreshToken{}, f.createRefreshTokenErr
	}
	return dbgen.RefreshToken{Token: arg.Token, UserID: arg.UserID, ExpiresAt: arg.ExpiresAt}, nil
}

func (f *fakeInvitationStore) GetScheme(context.Context, uuid.UUID) (dbgen.Scheme, error) {
	if f.schemeErr != nil {
		return dbgen.Scheme{}, f.schemeErr
	}
	if f.scheme == nil {
		return dbgen.Scheme{}, nil
	}
	return *f.scheme, nil
}

func (f *fakeInvitationStore) GetUnit(context.Context, uuid.UUID) (dbgen.Unit, error) {
	if f.unitErr != nil {
		return dbgen.Unit{}, f.unitErr
	}
	if f.unit == nil {
		return dbgen.Unit{}, nil
	}
	return *f.unit, nil
}

func (f *fakeInvitationStore) CreateUser(_ context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
	if f.createUserErr != nil {
		return dbgen.User{}, f.createUserErr
	}
	if f.createdUser == nil || f.createdUser.ID == uuid.Nil {
		f.createdUser = &dbgen.User{
			ID:       uuid.New(),
			Email:    arg.Email,
			FullName: arg.FullName,
		}
	}
	return *f.createdUser, nil
}

func (f *fakeInvitationStore) CreateOrgMembership(context.Context, dbgen.CreateOrgMembershipParams) (dbgen.OrgMembership, error) {
	return dbgen.OrgMembership{}, nil
}

func (f *fakeInvitationStore) UpsertSchemeMembership(context.Context, dbgen.UpsertSchemeMembershipParams) (dbgen.SchemeMembership, error) {
	return dbgen.SchemeMembership{}, nil
}

func newTestService(store *fakeInvitationStore) *Service {
	return &Service{
		q:             store,
		withTx:        func(ctx context.Context, fn func(q txStore) error) error { return fn(store) },
		sender:        &notification.NoopSender{},
		jwtSecret:     "unit-test-secret",
		jwtExpiry:     15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

func TestServiceCreateRejectsForeignScheme(t *testing.T) {
	orgID := uuid.New()
	store := &fakeInvitationStore{
		scheme: &dbgen.Scheme{
			ID:    uuid.New(),
			OrgID: uuid.New(),
		},
	}
	svc := newTestService(store)

	_, err := svc.Create(context.Background(), orgID.String(), CreateParams{
		Email:    "user@example.com",
		FullName: "Test User",
		Role:     "trustee",
		SchemeID: store.scheme.ID.String(),
	}, "http://localhost:3000")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
	if store.createdInvitation != nil {
		t.Fatal("Create() created invitation for foreign scheme")
	}
}

func TestServiceCreateRejectsUnitOutsideScheme(t *testing.T) {
	orgID := uuid.New()
	schemeID := uuid.New()
	store := &fakeInvitationStore{
		scheme: &dbgen.Scheme{
			ID:    schemeID,
			OrgID: orgID,
		},
		unit: &dbgen.Unit{
			ID:       uuid.New(),
			SchemeID: uuid.New(),
		},
	}
	svc := newTestService(store)

	_, err := svc.Create(context.Background(), orgID.String(), CreateParams{
		Email:    "resident@example.com",
		FullName: "Resident User",
		Role:     "resident",
		SchemeID: schemeID.String(),
		UnitID:   store.unit.ID.String(),
	}, "http://localhost:3000")
	if err == nil || err.Error() != "invalid unit_id" {
		t.Fatalf("Create() error = %v, want invalid unit_id", err)
	}
	if store.createdInvitation != nil {
		t.Fatal("Create() created invitation for unit outside scheme")
	}
}

func TestServiceCreateRecordsAuditBeforeInvitationSendFailure(t *testing.T) {
	orgID := uuid.New()
	schemeID := uuid.New()
	store := &fakeInvitationStore{
		scheme: &dbgen.Scheme{
			ID:    schemeID,
			OrgID: orgID,
		},
	}
	sender := &fakeInvitationSender{sendErr: errors.New("provider unavailable")}
	auditor := &fakeInvitationAuditor{}
	svc := &Service{
		q:             store,
		withTx:        func(ctx context.Context, fn func(q txStore) error) error { return fn(store) },
		sender:        sender,
		auditor:       auditor,
		jwtSecret:     "unit-test-secret",
		jwtExpiry:     15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}

	_, err := svc.Create(context.Background(), orgID.String(), CreateParams{
		Email:    "user@example.com",
		FullName: "Test User",
		Role:     "trustee",
		SchemeID: schemeID.String(),
	}, "http://localhost:3000")
	if err == nil {
		t.Fatal("Create() error = nil, want send failure")
	}
	if sender.calls != 1 {
		t.Fatalf("SendInvitation calls = %d, want 1", sender.calls)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	if auditor.events[0].Action != "invitation.created" {
		t.Fatalf("audit action = %q, want invitation.created", auditor.events[0].Action)
	}
}

func TestServiceResendRecordsAuditBeforeInvitationSendFailure(t *testing.T) {
	orgID := uuid.New()
	invitationID := uuid.New()
	schemeID := uuid.New()
	store := &fakeInvitationStore{
		invitationByID: &dbgen.Invitation{
			ID:        invitationID,
			OrgID:     orgID,
			SchemeID:  schemeID,
			Email:     "user@example.com",
			FullName:  "Test User",
			Role:      "trustee",
			Status:    "pending",
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	sender := &fakeInvitationSender{sendErr: errors.New("provider unavailable")}
	auditor := &fakeInvitationAuditor{}
	svc := &Service{
		q:             store,
		withTx:        func(ctx context.Context, fn func(q txStore) error) error { return fn(store) },
		sender:        sender,
		auditor:       auditor,
		jwtSecret:     "unit-test-secret",
		jwtExpiry:     15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}

	_, err := svc.Resend(context.Background(), orgID.String(), invitationID.String(), "http://localhost:3000")
	if err == nil {
		t.Fatal("Resend() error = nil, want send failure")
	}
	if sender.calls != 1 {
		t.Fatalf("SendInvitation calls = %d, want 1", sender.calls)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	if auditor.events[0].Action != "invitation.resent" {
		t.Fatalf("audit action = %q, want invitation.resent", auditor.events[0].Action)
	}
}

func TestServiceAcceptStoresHashedRefreshToken(t *testing.T) {
	orgID := uuid.New()
	schemeID := uuid.New()
	store := &fakeInvitationStore{
		invitationByToken: &dbgen.Invitation{
			ID:        uuid.New(),
			OrgID:     orgID,
			SchemeID:  schemeID,
			UnitID:    pgtype.UUID{},
			Email:     "trustee@example.com",
			FullName:  "Trustee User",
			Role:      "trustee",
			Token:     "invite-token",
			Status:    "pending",
			ExpiresAt: time.Now().Add(time.Hour),
		},
		getUserByEmailErr: pgx.ErrNoRows,
		createdUser: &dbgen.User{
			ID:       uuid.New(),
			Email:    "trustee@example.com",
			FullName: "Trustee User",
		},
	}
	svc := newTestService(store)

	resp, err := svc.Accept(context.Background(), "invite-token", "StrongPassword!123")
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if resp.RefreshToken == "" {
		t.Fatal("Accept() returned empty refresh token")
	}
	if store.createdRefreshToken == nil {
		t.Fatal("Accept() did not persist a refresh token")
	}
	if store.createdRefreshToken.Token == resp.RefreshToken {
		t.Fatal("Accept() stored refresh token in plaintext")
	}
	wantHash := auth.HashRefreshToken(resp.RefreshToken)
	if store.createdRefreshToken.Token != wantHash {
		t.Fatalf("stored refresh token hash = %q, want %q", store.createdRefreshToken.Token, wantHash)
	}
	if store.updatedInviteStatus == nil || store.updatedInviteStatus.Status != "accepted" {
		t.Fatalf("invitation status update = %+v, want accepted", store.updatedInviteStatus)
	}
}

func TestInvitationCreatedAuditEvent(t *testing.T) {
	event := invitationCreatedAuditEvent(invitationAuditInput{
		OrgID:        "org-1",
		SchemeID:     "scheme-1",
		ActorRole:    "admin",
		InvitationID: "inv-1",
		Email:        "user@example.com",
		FullName:     "Test User",
		Role:         "trustee",
		UnitID:       "unit-1",
		Status:       "pending",
		ExpiresAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	if event.Action != "invitation.created" {
		t.Fatalf("action = %q, want invitation.created", event.Action)
	}
	if event.ResourceType != "invitation" {
		t.Fatalf("resource_type = %q, want invitation", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["email"] != "user@example.com" {
		t.Fatalf("after.email = %v, want user@example.com", after["email"])
	}
	if _, exists := after["token"]; exists {
		t.Fatal("after state must not contain token")
	}
	if after["unit_id"] != "unit-1" {
		t.Fatalf("after.unit_id = %v, want unit-1", after["unit_id"])
	}
}

func TestInvitationCreatedAuditEventOmitsEmptyUnitID(t *testing.T) {
	event := invitationCreatedAuditEvent(invitationAuditInput{
		OrgID:        "org-1",
		SchemeID:     "scheme-1",
		ActorRole:    "admin",
		InvitationID: "inv-1",
		Email:        "user@example.com",
		FullName:     "Test User",
		Role:         "trustee",
		UnitID:       "",
		Status:       "pending",
		ExpiresAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if _, exists := after["unit_id"]; exists {
		t.Fatal("after state should not contain unit_id when empty")
	}
}

func TestInvitationResentAuditEvent(t *testing.T) {
	event := invitationResentAuditEvent(invitationAuditInput{
		OrgID:        "org-1",
		SchemeID:     "scheme-1",
		ActorRole:    "admin",
		InvitationID: "inv-1",
		Email:        "user@example.com",
		FullName:     "Test User",
		Role:         "trustee",
		Status:       "pending",
		ExpiresAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	if event.Action != "invitation.resent" {
		t.Fatalf("action = %q, want invitation.resent", event.Action)
	}
}

func TestInvitationRevokedAuditEvent(t *testing.T) {
	event := invitationRevokedAuditEvent(invitationAuditInput{
		OrgID:        "org-1",
		SchemeID:     "scheme-1",
		ActorRole:    "admin",
		InvitationID: "inv-1",
		Email:        "user@example.com",
		FullName:     "Test User",
		Role:         "trustee",
		Status:       "revoked",
		ExpiresAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	if event.Action != "invitation.revoked" {
		t.Fatalf("action = %q, want invitation.revoked", event.Action)
	}
	if event.BeforeState != nil {
		t.Fatal("before state should be nil for revoke")
	}
}

var _ audit.ResourceEvent
