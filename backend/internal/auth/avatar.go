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

	"example.com/uptime/backend/internal/store"
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
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}
	if s.files == nil {
		return Profile{}, fmt.Errorf("file storage is not configured")
	}
	filename, err := s.files.Save("avatars", ext, data)
	if err != nil {
		return Profile{}, err
	}
	if err := s.store.UpdateUserAvatar(ctx, id, filename); err != nil {
		_ = s.files.Delete("avatars", filename)
		return Profile{}, err
	}
	if user.AvatarFile != "" {
		_ = s.files.Delete("avatars", user.AvatarFile)
	}
	return s.Profile(ctx, token)
}
