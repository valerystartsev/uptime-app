package auth_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/migrations"
	"example.com/uptime/backend/internal/store"
)

func TestServiceRegisterAndRefreshRotation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	dataStore, err := store.Open(databaseURL)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = dataStore.Close() })
	if err := migrations.Run(context.Background(), dataStore.DB()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := dataStore.DB().Exec("TRUNCATE refresh_sessions, users CASCADE").Error; err != nil {
		t.Fatalf("truncate auth tables: %v", err)
	}

	tokens := auth.NewTokenManager([]byte("01234567890123456789012345678901"), "test", "test", time.Minute)
	service := auth.NewService(dataStore, tokens, time.Hour)
	registered, err := service.Register(context.Background(), "USER@example.com", "password1")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := tokens.Validate(registered.AccessToken); err != nil {
		t.Fatalf("Validate() registered access token: %v", err)
	}
	profile, err := service.Profile(context.Background(), registered.AccessToken)
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	if profile.Email != "user@example.com" {
		t.Fatalf("Profile() email = %q, want %q", profile.Email, "user@example.com")
	}
	updated, err := service.UpdateProfile(context.Background(), registered.AccessToken, "  Ada Lovelace  ")
	if err != nil || updated.Name != "Ada Lovelace" {
		t.Fatalf("UpdateProfile() = %#v, %v", updated, err)
	}
	stored, err := service.Profile(context.Background(), registered.AccessToken)
	if err != nil || stored.Name != "Ada Lovelace" {
		t.Fatalf("Profile() after update = %#v, %v", stored, err)
	}
	if _, err := service.UpdateProfile(context.Background(), registered.AccessToken, strings.Repeat("a", 100)); err != nil {
		t.Fatalf("UpdateProfile() with 100 characters error = %v", err)
	}
	for _, name := range []string{" ", strings.Repeat("a", 101)} {
		if _, err := service.UpdateProfile(context.Background(), registered.AccessToken, name); !errors.Is(err, auth.ErrInvalidName) {
			t.Fatalf("UpdateProfile(%q) error = %v, want ErrInvalidName", name, err)
		}
	}

	refreshed, err := service.Refresh(context.Background(), registered.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.RefreshToken == registered.RefreshToken {
		t.Fatal("Refresh() did not rotate the refresh token")
	}
	if _, err := service.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Fatalf("reusing token error = %v, want ErrInvalidRefreshToken", err)
	}
	if _, err := service.Register(context.Background(), "user@example.com", "password1"); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("duplicate registration error = %v, want ErrEmailTaken", err)
	}
}
