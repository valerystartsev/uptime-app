package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/config"
)

const maxRequestBodyBytes = 1 << 20

type Authenticator interface {
	Register(ctx context.Context, email, password string) (auth.SessionResult, error)
	Login(ctx context.Context, email, password string) (auth.SessionResult, error)
	Refresh(ctx context.Context, rawToken string) (auth.SessionResult, error)
	Logout(ctx context.Context, rawToken string) error
	Profile(ctx context.Context, accessToken string) (auth.Profile, error)
	UpdateProfile(ctx context.Context, accessToken, name string) (auth.Profile, error)
}

type Controller struct {
	authenticator Authenticator
	cookieConfig  config.CookieConfig
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type profileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type updateProfileRequest struct {
	Name string `json:"name"`
}

func New(authenticator Authenticator, cookieConfig config.CookieConfig) *Controller {
	return &Controller{authenticator: authenticator, cookieConfig: cookieConfig}
}

func (h *Controller) Register(w http.ResponseWriter, r *http.Request) {
	credentials, ok := h.readCredentials(w, r)
	if !ok {
		return
	}
	result, err := h.authenticator.Register(r.Context(), credentials.Email, credentials.Password)
	if err != nil {
		handleAuthError(w, err, true)
		return
	}
	h.writeSession(w, http.StatusCreated, result)
}

func (h *Controller) Login(w http.ResponseWriter, r *http.Request) {
	credentials, ok := h.readCredentials(w, r)
	if !ok {
		return
	}
	result, err := h.authenticator.Login(r.Context(), credentials.Email, credentials.Password)
	if err != nil {
		handleAuthError(w, err, false)
		return
	}
	h.writeSession(w, http.StatusOK, result)
}

func (h *Controller) Refresh(w http.ResponseWriter, r *http.Request) {
	result, err := h.authenticator.Refresh(r.Context(), h.refreshToken(r))
	if err != nil {
		handleAuthError(w, err, false)
		return
	}
	h.writeSession(w, http.StatusOK, result)
}

func (h *Controller) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.authenticator.Logout(r.Context(), h.refreshToken(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Controller) Me(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	profile, err := h.authenticator.Profile(r.Context(), accessToken)
	if err != nil {
		handleAuthError(w, err, false)
		return
	}
	writeProfile(w, profile)
}

func (h *Controller) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	var body updateProfileRequest
	if !readJSON(w, r, &body) {
		return
	}
	profile, err := h.authenticator.UpdateProfile(r.Context(), accessToken, body.Name)
	if err != nil {
		handleAuthError(w, err, false)
		return
	}
	writeProfile(w, profile)
}

func (h *Controller) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, auth.MaxAvatarBytes+1024)
	if err := r.ParseMultipartForm(1024); err != nil {
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	defer r.MultipartForm.RemoveAll()
	files := r.MultipartForm.File["avatar"]
	if len(files) != 1 || len(r.MultipartForm.File) != 1 || len(r.MultipartForm.Value) != 0 {
		writeError(w, http.StatusBadRequest, "expected one avatar file")
		return
	}
	file, err := files[0].Open()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, auth.MaxAvatarBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	uploader, ok := h.authenticator.(interface {
		UploadAvatar(context.Context, string, []byte) (auth.Profile, error)
	})
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	profile, err := uploader.UploadAvatar(r.Context(), token, data)
	if err != nil {
		handleAuthError(w, err, false)
		return
	}
	writeProfile(w, profile)
}

func writeProfile(w http.ResponseWriter, profile auth.Profile) {
	writeJSON(w, http.StatusOK, profileResponse{ID: profile.ID.String(), Email: profile.Email, Name: profile.Name, AvatarURL: profile.AvatarURL})
}

func (h *Controller) readCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	var credentials credentialsRequest
	if !readJSON(w, r, &credentials) {
		return credentialsRequest{}, false
	}
	return credentials, true
}

func readJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (h *Controller) writeSession(w http.ResponseWriter, status int, session auth.SessionResult) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieConfig.Name,
		Value:    session.RefreshToken,
		Path:     h.cookieConfig.Path,
		Domain:   h.cookieConfig.Domain,
		Expires:  session.RefreshExpiresAt,
		MaxAge:   maxAge(session.RefreshExpiresAt),
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: h.cookieConfig.SameSite,
	})
	writeJSON(w, status, tokenResponse{
		AccessToken: session.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(session.AccessTokenExpiry).Seconds()),
	})
}

func (h *Controller) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieConfig.Name,
		Path:     h.cookieConfig.Path,
		Domain:   h.cookieConfig.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: h.cookieConfig.SameSite,
	})
}

func (h *Controller) refreshToken(r *http.Request) string {
	cookie, err := r.Cookie(h.cookieConfig.Name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func handleAuthError(w http.ResponseWriter, err error, registration bool) {
	switch {
	case errors.Is(err, auth.ErrInvalidEmail), errors.Is(err, auth.ErrWeakPassword), errors.Is(err, auth.ErrInvalidName), errors.Is(err, auth.ErrInvalidAvatar):
		writeError(w, http.StatusBadRequest, err.Error())
	case registration && errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already registered")
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidRefreshToken), errors.Is(err, auth.ErrInvalidAccessToken):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || scheme != "Bearer" || token == "" {
		return "", false
	}
	return token, true
}

func maxAge(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
