package migrate_cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/initia-labs/core-indexer/pkg/db"
)

const (
	FlagDBHost        = "db-host"
	FlagDBUser        = "db-user"
	FlagMigrationsDir = "migrations-dir"
)

// MigrateCmd is the command for managing database migrations
func MigrateCmd() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
		Long:  "Manage database migrations using GORM and Atlas",
	}

	migrateCmd.AddCommand(
		generateCmd(),
	)

	return migrateCmd
}

// generateCmd generates a new migration file based on GORM models
func generateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [name]",
		Short: "Generate a new migration file",
		Long:  "Generate a new migration file based on GORM models",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbHost, _ := cmd.Flags().GetString(FlagDBHost)
			dbUser, _ := cmd.Flags().GetString(FlagDBUser)
			migrationsDir, _ := cmd.Flags().GetString(FlagMigrationsDir)
			name := args[0]

			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %w", err)
			}

			if err := db.GenerateMigrationFilesWithLivePostgres(dbHost, dbUser, name, migrationsDir); err != nil {
				return fmt.Errorf("failed to generate migration: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().String(FlagDBHost, "localhost", "Database host for local migration generation")
	cmd.Flags().String(FlagDBUser, "postgres", "Database user for local migration generation")
	cmd.Flags().String(FlagMigrationsDir, "db/migrations", "Directory to store migration files")

	return cmd
}
