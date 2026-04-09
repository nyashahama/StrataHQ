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
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

type fakeInvitationStore struct {
	scheme                *dbgen.Scheme
	unit                  *dbgen.Unit
	invitationByToken     *dbgen.Invitation
	createdUser           *dbgen.User
	createdInvitation     *dbgen.CreateInvitationParams
	createdRefreshToken   *dbgen.CreateRefreshTokenParams
	updatedInviteStatus   *dbgen.UpdateInvitationStatusParams
	schemeErr             error
	unitErr               error
	invitationByTokenErr  error
	getUserByEmailErr     error
	createUserErr         error
	createRefreshTokenErr error
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
	return dbgen.Invitation{}, pgx.ErrNoRows
}

func (f *fakeInvitationStore) UpdateInvitationToken(context.Context, dbgen.UpdateInvitationTokenParams) (dbgen.Invitation, error) {
	return dbgen.Invitation{}, nil
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
