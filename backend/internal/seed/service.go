package seed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

const (
	SeedDemoPasswordEnv = "SEED_DEMO_PASSWORD"
	DemoAdminEmail      = "agent@demo.stratahq.test"
	DemoTrusteeEmail    = "trustee@demo.stratahq.test"
	DemoResidentEmail   = "resident@demo.stratahq.test"
)

var demoUnits = []struct {
	Identifier       string
	OwnerName        string
	ResidentEmail    string
	ResidentFullName string
	ResidentRole     string
	Floor            int32
	SectionValueBps  int32
	ShouldAttachUser bool
}{
	{Identifier: "1A", OwnerName: "Henderson, T.", Floor: 1, SectionValueBps: 417},
	{Identifier: "2B", OwnerName: "Molefe, S.", Floor: 2, SectionValueBps: 417},
	{Identifier: "3A", OwnerName: "van der Berg, L.", Floor: 3, SectionValueBps: 417},
	{Identifier: "4B", OwnerName: "Naidoo, R.", Floor: 4, SectionValueBps: 417, ResidentEmail: DemoResidentEmail, ResidentFullName: "Riya Naidoo", ResidentRole: string(auth.RoleResident), ShouldAttachUser: true},
	{Identifier: "5A", OwnerName: "Khumalo, B.", Floor: 5, SectionValueBps: 417},
	{Identifier: "6C", OwnerName: "Abrahams, J.", Floor: 6, SectionValueBps: 417},
	{Identifier: "7B", OwnerName: "Petersen, M.", Floor: 7, SectionValueBps: 417},
	{Identifier: "8A", OwnerName: "Dlamini, S.", Floor: 8, SectionValueBps: 417},
}

type Service struct {
	db *database.Pool
}

type Result struct {
	OrgName        string
	OrgID          string
	SchemeName     string
	SchemeID       string
	AdminEmail     string
	TrusteeEmail   string
	ResidentEmail  string
	Password       string
	UnitsSeeded    int
	AlreadyExisted bool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) SeedDemo(ctx context.Context) (*Result, error) {
	adminUser, err := s.db.Q.GetUserByEmail(ctx, DemoAdminEmail)
	if err == nil {
		return s.describeExisting(ctx, adminUser.ID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	password, err := demoPassword()
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	result := &Result{
		OrgName:       "Demo Property Management",
		SchemeName:    "Sunridge Heights",
		UnitsSeeded:   len(demoUnits),
		AdminEmail:    DemoAdminEmail,
		TrusteeEmail:  DemoTrusteeEmail,
		ResidentEmail: DemoResidentEmail,
		Password:      password,
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		adminUser, txErr := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        DemoAdminEmail,
			PasswordHash: string(passwordHash),
			FullName:     "Demo Agent",
		})
		if txErr != nil {
			return txErr
		}

		org, txErr := q.CreateOrg(ctx, result.OrgName)
		if txErr != nil {
			return txErr
		}
		result.OrgID = org.ID.String()

		if _, txErr = q.UpdateOrg(ctx, dbgen.UpdateOrgParams{
			Name:         result.OrgName,
			ContactEmail: pgtype.Text{String: DemoAdminEmail, Valid: true},
			ID:           org.ID,
		}); txErr != nil {
			return txErr
		}

		if _, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: adminUser.ID,
			OrgID:  org.ID,
			Role:   string(auth.RoleAdmin),
		}); txErr != nil {
			return txErr
		}

		scheme, txErr := q.CreateScheme(ctx, dbgen.CreateSchemeParams{
			OrgID:     org.ID,
			Name:      result.SchemeName,
			Address:   "14 Sunridge Drive, Claremont, Cape Town, 7708",
			UnitCount: 24,
		})
		if txErr != nil {
			return txErr
		}
		result.SchemeID = scheme.ID.String()

		var residentUnitID pgtype.UUID
		for _, unit := range demoUnits {
			unitRow, unitErr := q.CreateUnit(ctx, dbgen.CreateUnitParams{
				SchemeID:        scheme.ID,
				Identifier:      unit.Identifier,
				OwnerName:       unit.OwnerName,
				Floor:           unit.Floor,
				SectionValueBps: unit.SectionValueBps,
			})
			if unitErr != nil {
				return unitErr
			}
			if unit.ShouldAttachUser {
				residentUnitID = pgtype.UUID{Bytes: unitRow.ID, Valid: true}
			}
		}

		trusteeUser, txErr := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        DemoTrusteeEmail,
			PasswordHash: string(passwordHash),
			FullName:     "Tina Trustee",
		})
		if txErr != nil {
			return txErr
		}
		if _, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: trusteeUser.ID,
			OrgID:  org.ID,
			Role:   string(auth.RoleTrustee),
		}); txErr != nil {
			return txErr
		}
		if _, txErr = q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
			UserID:   trusteeUser.ID,
			SchemeID: scheme.ID,
			UnitID:   pgtype.UUID{},
			Role:     string(auth.RoleTrustee),
		}); txErr != nil {
			return txErr
		}

		residentUser, txErr := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        DemoResidentEmail,
			PasswordHash: string(passwordHash),
			FullName:     "Riya Naidoo",
		})
		if txErr != nil {
			return txErr
		}
		if _, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: residentUser.ID,
			OrgID:  org.ID,
			Role:   string(auth.RoleResident),
		}); txErr != nil {
			return txErr
		}
		if _, txErr = q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
			UserID:   residentUser.ID,
			SchemeID: scheme.ID,
			UnitID:   residentUnitID,
			Role:     string(auth.RoleResident),
		}); txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) describeExisting(ctx context.Context, adminUserID uuid.UUID) (*Result, error) {
	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, errors.New("seeded admin user has no org memberships")
	}

	orgID := memberships[0].OrgID
	schemes, err := s.db.Q.ListSchemesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := &Result{
		AlreadyExisted: true,
		OrgName:        memberships[0].OrgName,
		OrgID:          orgID.String(),
		AdminEmail:     DemoAdminEmail,
		TrusteeEmail:   DemoTrusteeEmail,
		ResidentEmail:  DemoResidentEmail,
	}

	if len(schemes) > 0 {
		result.SchemeID = schemes[0].ID.String()
		result.SchemeName = schemes[0].Name
		units, unitErr := s.db.Q.ListUnitsByScheme(ctx, schemes[0].ID)
		if unitErr != nil {
			return nil, unitErr
		}
		result.UnitsSeeded = len(units)
	}

	return result, nil
}

func demoPassword() (string, error) {
	if password := os.Getenv(SeedDemoPasswordEnv); password != "" {
		return password, nil
	}

	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return "Demo-" + hex.EncodeToString(buf) + "!", nil
}
