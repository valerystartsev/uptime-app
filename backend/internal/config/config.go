package config

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const minimumJWTSecretLength = 32

type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	SameSite http.SameSite
}

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	JWTSecret         []byte
	JWTIssuer         string
	JWTAudience       string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	CORSAllowedOrigin string
	AvatarDir         string
	RefreshCookie     CookieConfig
}

func Load() (Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minimumJWTSecretLength {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d bytes", minimumJWTSecretLength)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	accessTTL, err := durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return Config{}, fmt.Errorf("token lifetimes must be positive")
	}

	secure, err := boolEnv("REFRESH_COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	sameSite, err := parseSameSite(envOrDefault("REFRESH_COOKIE_SAME_SITE", "lax"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:          envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:       databaseURL,
		JWTSecret:         []byte(jwtSecret),
		JWTIssuer:         envOrDefault("JWT_ISSUER", "uptime-backend"),
		JWTAudience:       envOrDefault("JWT_AUDIENCE", "uptime-frontend"),
		AccessTokenTTL:    accessTTL,
		RefreshTokenTTL:   refreshTTL,
		CORSAllowedOrigin: os.Getenv("CORS_ALLOWED_ORIGIN"),
		AvatarDir:         envOrDefault("AVATAR_DIR", "uploads/avatars"),
		RefreshCookie: CookieConfig{
			Name:     envOrDefault("REFRESH_COOKIE_NAME", "refresh_token"),
			Domain:   os.Getenv("REFRESH_COOKIE_DOMAIN"),
			Path:     envOrDefault("REFRESH_COOKIE_PATH", "/"),
			Secure:   secure,
			SameSite: sameSite,
		},
	}, nil
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return duration, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}

func parseSameSite(value string) (http.SameSite, error) {
	switch strings.ToLower(value) {
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return 0, fmt.Errorf("invalid REFRESH_COOKIE_SAME_SITE %q", value)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
