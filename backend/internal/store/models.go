package store

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"not null"`
	Name         string    `gorm:"not null"`
	AvatarFile   string    `gorm:"not null"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (User) TableName() string {
	return "users"
}

type RefreshSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	TokenHash []byte    `gorm:"type:bytea;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID"`
}

func (RefreshSession) TableName() string {
	return "refresh_sessions"
}
