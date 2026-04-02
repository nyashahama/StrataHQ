package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/seed"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	result, err := seed.NewService(db).SeedDemo(ctx)
	if err != nil {
		logger.Error("failed to seed demo data", "error", err)
		os.Exit(1)
	}

	state := "created"
	if result.AlreadyExisted {
		state = "already present"
	}

	fmt.Printf("Demo seed %s\n", state)
	fmt.Printf("Org: %s (%s)\n", result.OrgName, result.OrgID)
	fmt.Printf("Scheme: %s (%s)\n", result.SchemeName, result.SchemeID)
	fmt.Printf("Units seeded: %d\n", result.UnitsSeeded)
	fmt.Printf("Admin login: %s / %s\n", result.AdminEmail, result.Password)
	fmt.Printf("Trustee login: %s / %s\n", result.TrusteeEmail, result.Password)
	fmt.Printf("Resident login: %s / %s\n", result.ResidentEmail, result.Password)
}
