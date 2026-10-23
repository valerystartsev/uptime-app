package auth

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"example.com/uptime/backend/internal/files"
	"example.com/uptime/backend/internal/store"
)

var ErrEmailTaken = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrInvalidAccessToken = errors.New("invalid access token")
var ErrInvalidName = errors.New("name must contain 1 to 100 characters")

type SessionResult struct {
	AccessToken       string
	AccessTokenExpiry time.Time
	RefreshToken      string
	RefreshExpiresAt  time.Time
}

type Profile struct {
	ID        uuid.UUID
	Email     string
	Name      string
	AvatarURL string
}

type Service struct {
	store           *store.Store
	tokens          *TokenManager
	refreshTokenTTL time.Duration
	now             func() time.Time
	files           *files.Storage
}

func NewService(dataStore *store.Store, tokens *TokenManager, refreshTokenTTL time.Duration) *Service {
	return &Service{store: dataStore, tokens: tokens, refreshTokenTTL: refreshTokenTTL, now: time.Now}
}

func (s *Service) SetFileStorage(storage *files.Storage) { s.files = storage }

func (s *Service) Register(ctx context.Context, email, password string) (SessionResult, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return SessionResult{}, err
	}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return SessionResult{}, err
	}
	user := store.User{ID: uuid.New(), Email: normalizedEmail, PasswordHash: passwordHash, CreatedAt: s.now().UTC()}
	result, err := s.createSession(ctx, &user, true)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return SessionResult{}, ErrEmailTaken
	}
	return result, err
}

func (s *Service) Login(ctx context.Context, email, password string) (SessionResult, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return SessionResult{}, ErrInvalidCredentials
	}
	user, err := s.store.FindUserByEmail(ctx, normalizedEmail)
	if errors.Is(err, store.ErrNotFound) {
		return SessionResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResult{}, err
	}
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return SessionResult{}, ErrInvalidCredentials
	}
	return s.createSession(ctx, user, false)
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (SessionResult, error) {
	if rawToken == "" {
		return SessionResult{}, ErrInvalidRefreshToken
	}
	now := s.now().UTC()
	hash := HashRefreshToken(rawToken)
	var user *store.User
	var result SessionResult
	err := s.store.WithinTransaction(ctx, func(tx *store.Store) error {
		session, err := tx.FindActiveSessionForUpdate(ctx, hash, now)
		if errors.Is(err, store.ErrNotFound) {
			return ErrInvalidRefreshToken
		}
		if err != nil {
			return err
		}
		if err := tx.RevokeRefreshSession(ctx, session.ID, now); err != nil {
			return err
		}
		user = &session.User
		result, err = s.newSession(ctx, tx, user, now)
		return err
	})
	if err != nil {
		return SessionResult{}, err
	}
	accessToken, accessExpiry, err := s.tokens.Issue(user.ID, user.Email)
	if err != nil {
		return SessionResult{}, err
	}
	result.AccessToken = accessToken
	result.AccessTokenExpiry = accessExpiry
	return result, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	err := s.store.WithinTransaction(ctx, func(tx *store.Store) error {
		session, err := tx.FindActiveSessionForUpdate(ctx, HashRefreshToken(rawToken), s.now().UTC())
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.RevokeRefreshSession(ctx, session.ID, s.now().UTC())
	})
	return err
}

func (s *Service) Profile(ctx context.Context, accessToken string) (Profile, error) {
	userID, err := s.userIDFromToken(accessToken)
	if err != nil {
		return Profile{}, err
	}
	user, err := s.store.FindUserByID(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return Profile{}, ErrInvalidAccessToken
	}
	if err != nil {
		return Profile{}, err
	}
	profile := Profile{ID: user.ID, Email: user.Email, Name: user.Name}
	if user.AvatarFile != "" {
		profile.AvatarURL = "/api/v1/avatars/" + user.AvatarFile
	}
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, accessToken, name string) (Profile, error) {
	userID, err := s.userIDFromToken(accessToken)
	if err != nil {
		return Profile{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return Profile{}, ErrInvalidName
	}
	if err := s.store.UpdateUserName(ctx, userID, name); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Profile{}, ErrInvalidAccessToken
		}
		return Profile{}, err
	}
	return s.Profile(ctx, accessToken)
}

func (s *Service) userIDFromToken(accessToken string) (uuid.UUID, error) {
	claims, err := s.tokens.Validate(accessToken)
	if err != nil {
		return uuid.Nil, ErrInvalidAccessToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidAccessToken
	}
	return userID, nil
}

func (s *Service) createSession(ctx context.Context, user *store.User, createUser bool) (SessionResult, error) {
	var result SessionResult
	err := s.store.WithinTransaction(ctx, func(tx *store.Store) error {
		if createUser {
			if err := tx.CreateUser(ctx, user); err != nil {
				return err
			}
		}
		var err error
		result, err = s.newSession(ctx, tx, user, s.now().UTC())
		return err
	})
	if err != nil {
		return SessionResult{}, err
	}
	accessToken, accessExpiry, err := s.tokens.Issue(user.ID, user.Email)
	if err != nil {
		return SessionResult{}, err
	}
	result.AccessToken = accessToken
	result.AccessTokenExpiry = accessExpiry
	return result, nil
}

func (s *Service) newSession(ctx context.Context, tx *store.Store, user *store.User, now time.Time) (SessionResult, error) {
	rawToken, tokenHash, err := NewRefreshToken()
	if err != nil {
		return SessionResult{}, err
	}
	expiresAt := now.Add(s.refreshTokenTTL)
	session := store.RefreshSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	if err := tx.CreateRefreshSession(ctx, &session); err != nil {
		return SessionResult{}, err
	}
	return SessionResult{RefreshToken: rawToken, RefreshExpiresAt: expiresAt}, nil
}
