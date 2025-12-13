package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all .up.sql migrations from the specified directory
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsPath string) error {
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}

	sort.Strings(upFiles)

	for _, filename := range upFiles {
		path := filepath.Join(migrationsPath, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading migration file %s: %w", filename, err)
		}

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", filename, err)
		}
	}

	return nil
}
