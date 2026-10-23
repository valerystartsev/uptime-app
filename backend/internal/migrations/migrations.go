package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

//go:embed sql/*.up.sql
var files embed.FS

const migrationsTable = "auth_schema_migrations"

func Run(ctx context.Context, db *gorm.DB) error {
	entries, err := fs.ReadDir(files, "sql")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("CREATE TABLE IF NOT EXISTS " + migrationsTable + " (version BIGINT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())").Error; err != nil {
			return fmt.Errorf("create migrations table: %w", err)
		}
		for _, entry := range entries {
			version, err := migrationVersion(entry.Name())
			if err != nil {
				return err
			}
			var count int64
			if err := tx.Table(migrationsTable).Where("version = ?", version).Count(&count).Error; err != nil {
				return fmt.Errorf("check migration %d: %w", version, err)
			}
			if count > 0 {
				continue
			}
			sql, err := files.ReadFile("sql/" + entry.Name())
			if err != nil {
				return fmt.Errorf("read migration %s: %w", entry.Name(), err)
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
			}
			if err := tx.Table(migrationsTable).Create(map[string]any{"version": version}).Error; err != nil {
				return fmt.Errorf("record migration %d: %w", version, err)
			}
		}
		return nil
	})
}

func migrationVersion(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("invalid migration name %q", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %q: %w", name, err)
	}
	return version, nil
}
