package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ariga.io/atlas-provider-gorm/gormschema"
	"ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	golangmigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

const (
	AdminDBName string = "postgres"
	SchemaName  string = "public"

	TempGormModelsSQL string = "gorm_models.sql"
	CustomTypesSQL    string = "custom_types.sql"
)

func GenerateMigrationFilesWithLivePostgres(pgHost, pgUser, name, migrationDir string) error {
	ctx := context.Background()

	shadowDB := fmt.Sprintf("atlas_shadow_%d", time.Now().UnixNano())
	dsnAdmin := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", pgHost, pgUser, AdminDBName)
	dsnShadow := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", pgHost, pgUser, shadowDB)

	adminConn, err := sql.Open("postgres", dsnAdmin)
	if err != nil {
		return fmt.Errorf("could not connect to admin db: %w", err)
	}
	defer adminConn.Close()

	_, err = adminConn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", shadowDB))
	if err != nil {
		return fmt.Errorf("could not create database %s: %w", shadowDB, err)
	}
	defer adminConn.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", shadowDB))

	shadowConn, err := sql.Open("postgres", dsnShadow)
	if err != nil {
		return fmt.Errorf("could not connect to shadow db: %w", err)
	}
	defer shadowConn.Close()

	drv, err := atlaspostgres.Open(shadowConn)
	if err != nil {
		return fmt.Errorf("could not connect to shadow postgres: %w", err)
	}

	dir, err := migrate.NewLocalDir(migrationDir)
	if err != nil {
		return fmt.Errorf("could not open migration dir: %w", err)
	}

	exec, err := migrate.NewExecutor(drv, dir, migrate.NopRevisionReadWriter{})
	if err != nil {
		return fmt.Errorf("could not create new migration executor: %w", err)
	}

	if err = executeUpMigrationFiles(ctx, dir, exec); err != nil {
		return fmt.Errorf("could not apply migrations: %w", err)
	}

	current, err := drv.InspectSchema(ctx, SchemaName, nil)
	if err != nil {
		return fmt.Errorf("could not inspect schema: %w", err)
	}

	gormDBName := fmt.Sprintf("gorm_model_%d", time.Now().UnixNano())
	gormDsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", pgHost, pgUser, gormDBName)

	_, err = adminConn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", gormDBName))
	if err != nil {
		return fmt.Errorf("could not create database %s: %w", shadowDB, err)
	}
	defer func() {
		_, err := adminConn.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", gormDBName))
		if err != nil {
			fmt.Println(err)
		}
	}()

	gormConn, err := sql.Open("postgres", gormDsn)
	if err != nil {
		return fmt.Errorf("could not connect to gorm db: %w", err)
	}
	defer gormConn.Close()

	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{DSN: gormDsn}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("could not connect to gorm db: %w", err)
	}
	defer func() {
		db, err := gormDB.DB()
		if err == nil {
			db.Close()
		}
	}()

	loader := gormschema.New("postgres", gormschema.WithConfig(gormDB.Config))
	srcDDL, err := loader.Load(AllModels...)
	if err != nil {
		return fmt.Errorf("could not load all models: %w", err)
	}

	memDrv, err := atlaspostgres.Open(gormConn)
	if err != nil {
		return fmt.Errorf("could not connect to gorm postgres: %w", err)
	}

	td, err := os.MkdirTemp("", "atlas-gorm-*")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer func() {
		err := os.RemoveAll(td)
		if err != nil {
			fmt.Printf("could not remove %s, manually remove it", td)
		}
	}()

	tmpDir, err := migrate.NewLocalDir(td)
	if err != nil {
		return fmt.Errorf("could not create temporary local dir: %w", err)
	}

	customTypes, err := customTypesSQLFile.Open(CustomTypesSQL)
	if err != nil {
		return fmt.Errorf("could not open embedded custom types: %w", err)
	}
	defer customTypes.Close()

	customTypesContent, err := io.ReadAll(customTypes)
	if err != nil {
		return fmt.Errorf("could not read embedded custom types: %w", err)
	}

	if err := tmpDir.WriteFile(CustomTypesSQL, customTypesContent); err != nil {
		return fmt.Errorf("could not write embedded custom types: %w", err)
	}

	if err := tmpDir.WriteFile(TempGormModelsSQL, []byte(srcDDL)); err != nil {
		return fmt.Errorf("could not write temporary file: %w", err)
	}

	memExec, err := migrate.NewExecutor(memDrv, tmpDir, migrate.NopRevisionReadWriter{})
	if err != nil {
		return fmt.Errorf("could not create new migration executor: %w", err)
	}

	tmpFiles, err := tmpDir.Files()
	if err != nil {
		return fmt.Errorf("could not get temp files: %w", err)
	}
	if err := memExec.ExecuteFiles(ctx, tmpFiles); err != nil {
		return fmt.Errorf("could not execute migration: %w", err)
	}

	desired, err := memDrv.InspectSchema(ctx, SchemaName, nil)
	if err != nil {
		return fmt.Errorf("could not inspect schema: %w", err)
	}

	planner := migrate.NewPlanner(drv, dir)
	planName := fmt.Sprintf("%s.up", name)

	changes, err := drv.SchemaDiff(current, desired)
	if err != nil {
		return fmt.Errorf("schema diff: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("No schema changes detected.")
		return nil
	}

	plan, err := drv.PlanChanges(ctx, planName, changes)
	if err != nil {
		return fmt.Errorf("plan changes: %w", err)
	}

	if err := planner.WritePlan(plan); err != nil {
		return fmt.Errorf("write migration: %w", err)
	}

	fmt.Println("Migration file generated:", planName)
	return nil
}

func executeUpMigrationFiles(ctx context.Context, migrationDir *migrate.LocalDir, executor *migrate.Executor) error {
	files, err := migrationDir.Files()
	if err != nil {
		return fmt.Errorf("read migration files: %w", err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	var result []migrate.File
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			result = append(result, f)
		}
	}

	return executor.ExecuteFiles(ctx, result)
}

// ApplyMigrationFiles applies pending migrations to the database using golang-migrate.
// If a migration fails (e.g. because an earlier pg_dump migration cleared search_path),
// it retries on a fresh connection with search_path explicitly set to "public".
func ApplyMigrationFiles(logger *zerolog.Logger, dbClient *gorm.DB, migrationsDir string) error {
	sqlDB, err := dbClient.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsDir)

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := golangmigrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err == nil || errors.Is(err, golangmigrate.ErrNoChange) {
		if errors.Is(err, golangmigrate.ErrNoChange) {
			logger.Info().Msg("No pending migrations to apply")
		} else {
			logger.Info().Msg("Successfully applied all pending migrations")
		}
		return nil
	}

	logger.Warn().Err(err).Msg("Migration failed, retrying with reset search_path")

	return retryMigrationsWithSearchPath(logger, sqlDB, sourceURL, migrationsDir)
}

func retryMigrationsWithSearchPath(logger *zerolog.Logger, sqlDB *sql.DB, sourceURL, migrationsDir string) error {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		retryDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{SchemaName: SchemaName})
		if err != nil {
			return fmt.Errorf("retry: failed to create postgres driver: %w", err)
		}

		version, dirty, err := retryDriver.Version()
		if err != nil {
			return fmt.Errorf("retry: failed to get migration version: %w", err)
		}

		if dirty {
			filePath := findUpMigrationFile(migrationsDir, version)
			if filePath == "" {
				return fmt.Errorf("retry: dirty migration version %d but no .up.sql file found", version)
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("retry: failed to read migration file %s: %w", filePath, err)
			}
			if err := retryDriver.Run(strings.NewReader(string(content))); err != nil {
				return fmt.Errorf("retry: failed to re-run migration %d: %w", version, err)
			}
			if err := retryDriver.SetVersion(version, false); err != nil {
				return fmt.Errorf("retry: failed to set version %d: %w", version, err)
			}
			logger.Info().Int("version", version).Msg("Re-applied dirty migration with reset search_path")
		}

		m, err := golangmigrate.NewWithDatabaseInstance(sourceURL, "postgres", retryDriver)
		if err != nil {
			return fmt.Errorf("retry: failed to create migrate instance: %w", err)
		}

		err = m.Up()

		if err == nil || errors.Is(err, golangmigrate.ErrNoChange) {
			logger.Info().Msg("Successfully applied all pending migrations")
			return nil
		}

		logger.Warn().Err(err).Int("attempt", i+1).Msg("Migration still failing, retrying")
	}

	return fmt.Errorf("failed to apply migrations after %d retries", maxRetries)
}

func findUpMigrationFile(dir string, version int) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	prefix := fmt.Sprintf("%d_", version)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".up.sql") {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}
