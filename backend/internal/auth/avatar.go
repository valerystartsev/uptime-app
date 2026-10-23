package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"

	"example.com/uptime/backend/internal/store"
	"github.com/google/uuid"
)

const MaxAvatarBytes = 5 << 20

var ErrInvalidAvatar = errors.New("avatar must be a JPEG or PNG image up to 5 MB and 4096 pixels per side")

func (s *Service) UploadAvatar(ctx context.Context, token string, data []byte) (Profile, error) {
	id, err := s.userIDFromToken(token)
	if err != nil {
		return Profile{}, err
	}
	user, err := s.store.FindUserByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return Profile{}, ErrInvalidAccessToken
	}
	if err != nil {
		return Profile{}, err
	}
	if len(data) == 0 || len(data) > MaxAvatarBytes {
		return Profile{}, ErrInvalidAvatar
	}
	mime := http.DetectContentType(data)
	if mime != "image/jpeg" && mime != "image/png" {
		return Profile{}, ErrInvalidAvatar
	}
	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 || imageConfig.Width > 4096 || imageConfig.Height > 4096 {
		return Profile{}, ErrInvalidAvatar
	}
	if (mime == "image/jpeg" && format != "jpeg") || (mime == "image/png" && format != "png") {
		return Profile{}, ErrInvalidAvatar
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return Profile{}, ErrInvalidAvatar
	}
	if err := os.MkdirAll(s.avatarDir, 0700); err != nil {
		return Profile{}, fmt.Errorf("create avatar directory: %w", err)
	}
	tmp, err := os.CreateTemp(s.avatarDir, ".avatar-*")
	if err != nil {
		return Profile{}, fmt.Errorf("create avatar file: %w", err)
	}
	defer os.Remove(tmp.Name())
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}
	filename := uuid.NewString() + ext
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Profile{}, err
	}
	if err := tmp.Close(); err != nil {
		return Profile{}, err
	}
	path := filepath.Join(s.avatarDir, filename)
	if err := os.Rename(tmp.Name(), path); err != nil {
		return Profile{}, err
	}
	if err := s.store.UpdateUserAvatar(ctx, id, filename); err != nil {
		os.Remove(path)
		return Profile{}, err
	}
	if user.AvatarFile != "" {
		_ = os.Remove(filepath.Join(s.avatarDir, user.AvatarFile))
	}
	return s.Profile(ctx, token)
}
