package migrations_test

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/uptime/backend/internal/migrations"
)

func TestRunCreatesAuthTables(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := migrations.Run(context.Background(), db); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, table := range []string{"users", "refresh_sessions"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("migration did not create %q", table)
		}
	}
	if !db.Migrator().HasColumn("users", "name") {
		t.Fatal("migration did not add users.name")
	}
}
