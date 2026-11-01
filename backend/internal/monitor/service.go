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
var ErrMonitorLimit = errors.New("monitor limit reached")
var ErrMonitorExists = errors.New("monitor already exists")
var ErrInvalidPage = errors.New("page must be 1 or 2")

const PageSize = store.MonitorPageSize
const MaxPages = store.MaxMonitors / PageSize

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

func (s *Service) List(ctx context.Context, token string, page int) ([]store.Monitor, bool, error) {
	if page < 1 || page > MaxPages {
		return nil, false, ErrInvalidPage
	}
	profile, err := s.auth.Profile(ctx, token)
	if err != nil {
		return nil, false, err
	}
	monitors, err := s.store.ListMonitors(ctx, profile.ID, PageSize, (page-1)*PageSize)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(monitors) > PageSize
	if hasMore {
		monitors = monitors[:PageSize]
	}
	return monitors, hasMore, nil
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
