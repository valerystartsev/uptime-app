package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManagerIssueAndValidate(t *testing.T) {
	manager := NewTokenManager([]byte("01234567890123456789012345678901"), "issuer", "audience", time.Minute)
	userID := uuid.New()
	token, expiresAt, err := manager.Issue(userID, "user@example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("Issue() expiry = %v, want future time", expiresAt)
	}
	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.Subject != userID.String() || claims.Email != "user@example.com" {
		t.Fatalf("Validate() claims = %#v", claims)
	}
}

func TestTokenManagerRejectsWrongSecret(t *testing.T) {
	issuer := NewTokenManager([]byte("01234567890123456789012345678901"), "issuer", "audience", time.Minute)
	token, _, err := issuer.Issue(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	validator := NewTokenManager([]byte("abcdefghijklmnopqrstuvwxyz123456"), "issuer", "audience", time.Minute)
	if _, err := validator.Validate(token); err == nil {
		t.Fatal("Validate() accepted token signed with another secret")
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager := NewTokenManager([]byte("01234567890123456789012345678901"), "issuer", "audience", -time.Minute)
	token, _, err := manager.Issue(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := manager.Validate(token); err == nil {
		t.Fatal("Validate() accepted an expired token")
	}
}

func TestRefreshTokensAreUniqueAndHashable(t *testing.T) {
	first, firstHash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	second, secondHash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	if first == second {
		t.Fatal("NewRefreshToken() returned duplicate values")
	}
	if string(firstHash) != string(HashRefreshToken(first)) {
		t.Fatal("HashRefreshToken() differs from generated hash")
	}
	if string(firstHash) == string(secondHash) {
		t.Fatal("different refresh tokens have identical hashes")
	}
}
