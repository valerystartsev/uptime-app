package monitor

import (
	"context"
	"encoding/base64"
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
var ErrMonitorLimit = errors.New("monitor limit reached")
var ErrMonitorExists = errors.New("monitor already exists")
var ErrInvalidCursor = errors.New("invalid cursor")

const PageSize = store.MonitorPageSize

const maxSafeIntervalValue = 1<<53 - 1 // JSON numbers must remain exact in the frontend.

type Service struct {
	store *store.Store
	auth  *auth.Service
}

func NewService(dataStore *store.Store, authenticator *auth.Service) *Service {
	return &Service{store: dataStore, auth: authenticator}
}

func (s *Service) Create(ctx context.Context, token, rawURL string, intervalValue int64, intervalUnit string) (store.Monitor, error) {
	profile, err := s.auth.Profile(ctx, token)
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
		ID: uuid.New(), UserID: profile.ID, URL: address,
		IntervalValue: intervalValue, IntervalUnit: intervalUnit, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateMonitor(ctx, &monitor); err != nil {
		if errors.Is(err, store.ErrMonitorExists) {
			return store.Monitor{}, ErrMonitorExists
		}
		if errors.Is(err, store.ErrMonitorLimit) {
			return store.Monitor{}, ErrMonitorLimit
		}
		return store.Monitor{}, err
	}
	return monitor, nil
}

func validInterval(value int64, unit string) bool {
	return value > 0 && value <= maxSafeIntervalValue &&
		(unit == "seconds" || unit == "minutes" || unit == "hours")
}

func (s *Service) List(ctx context.Context, token string, cursor string) ([]store.Monitor, string, error) {
	boundary, err := parseCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	profile, err := s.auth.Profile(ctx, token)
	if err != nil {
		return nil, "", err
	}
	monitors, err := s.store.ListMonitors(ctx, profile.ID, PageSize, boundary)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(monitors) > PageSize {
		monitors = monitors[:PageSize]
		last := monitors[len(monitors)-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID.String()))
	}
	return monitors, next, nil
}

func parseCursor(value string) (*store.MonitorCursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 128 {
		return nil, ErrInvalidCursor
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	timestamp, id, ok := strings.Cut(string(decoded), "|")
	createdAt, timeErr := time.Parse(time.RFC3339Nano, timestamp)
	monitorID, idErr := uuid.Parse(id)
	if !ok || timeErr != nil || idErr != nil {
		return nil, ErrInvalidCursor
	}
	return &store.MonitorCursor{CreatedAt: createdAt, ID: monitorID}, nil
}

func validURL(value string) bool {
	if len(value) == 0 || len(value) > 2048 || strings.IndexFunc(value, func(r rune) bool {
		return r == '\\' || r < 0x20 || r == 0x7f
	}) >= 0 {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == "" && parsed.Fragment == ""
}
