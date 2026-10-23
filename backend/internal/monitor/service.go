package monitor

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/store"
)

var ErrInvalidURL = errors.New("URL must be an HTTP or HTTPS address up to 2048 characters")
var ErrInvalidInterval = errors.New("interval must be a positive number of seconds, minutes, or hours")

const maxSafeIntervalValue = 1<<53 - 1 // JSON numbers must remain exact in the frontend.

type Service struct {
	store  *store.Store
	tokens *auth.TokenManager
}

func NewService(dataStore *store.Store, tokens *auth.TokenManager) *Service {
	return &Service{store: dataStore, tokens: tokens}
}

func (s *Service) Create(ctx context.Context, token, rawURL string, intervalValue int64, intervalUnit string) (store.Monitor, error) {
	userID, err := s.userID(ctx, token)
	if err != nil {
		return store.Monitor{}, err
	}
	address := strings.TrimSpace(rawURL)
	if !validURL(address) {
		return store.Monitor{}, ErrInvalidURL
	}
	if !validInterval(intervalValue, intervalUnit) {
		return store.Monitor{}, ErrInvalidInterval
	}
	monitor := store.Monitor{
		ID: uuid.New(), UserID: userID, URL: address,
		IntervalValue: intervalValue, IntervalUnit: intervalUnit, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateMonitor(ctx, &monitor); err != nil {
		return store.Monitor{}, err
	}
	return monitor, nil
}

func validInterval(value int64, unit string) bool {
	return value > 0 && value <= maxSafeIntervalValue &&
		(unit == "seconds" || unit == "minutes" || unit == "hours")
}

func (s *Service) List(ctx context.Context, token string) ([]store.Monitor, error) {
	userID, err := s.userID(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.store.ListMonitors(ctx, userID)
}

func (s *Service) userID(ctx context.Context, token string) (uuid.UUID, error) {
	claims, err := s.tokens.Validate(token)
	if err != nil {
		return uuid.Nil, auth.ErrInvalidAccessToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, auth.ErrInvalidAccessToken
	}
	if _, err := s.store.FindUserByID(ctx, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return uuid.Nil, auth.ErrInvalidAccessToken
		}
		return uuid.Nil, err
	}
	return userID, nil
}

func validURL(value string) bool {
	if len(value) == 0 || len(value) > 2048 {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == "" && parsed.Fragment == ""
}
