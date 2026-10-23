package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("store: not found")

type Store struct {
	db *gorm.DB
}

func Open(databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("get sql database: %w", err)
	}
	return sqlDB.Close()
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(*Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}

func (s *Store) CreateUser(ctx context.Context, user *User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

func (s *Store) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func (s *Store) UpdateUserName(ctx context.Context, id uuid.UUID, name string) error {
	result := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("name", name)
	if result.Error != nil {
		return fmt.Errorf("update user name: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserAvatar(ctx context.Context, id uuid.UUID, filename string) error {
	result := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("avatar_file", filename)
	if result.Error != nil {
		return fmt.Errorf("update user avatar: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateMonitor(ctx context.Context, monitor *Monitor) error {
	if err := s.db.WithContext(ctx).Create(monitor).Error; err != nil {
		return fmt.Errorf("create monitor: %w", err)
	}
	return nil
}

func (s *Store) ListMonitors(ctx context.Context, userID uuid.UUID) ([]Monitor, error) {
	monitors := make([]Monitor, 0)
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").Find(&monitors).Error; err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	return monitors, nil
}

func (s *Store) CreateRefreshSession(ctx context.Context, session *RefreshSession) error {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

func (s *Store) FindActiveSessionForUpdate(ctx context.Context, tokenHash []byte, now time.Time) (*RefreshSession, error) {
	var session RefreshSession
	err := s.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh session: %w", err)
	}
	return &session, nil
}

func (s *Store) RevokeRefreshSession(ctx context.Context, id uuid.UUID, now time.Time) error {
	result := s.db.WithContext(ctx).Model(&RefreshSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
