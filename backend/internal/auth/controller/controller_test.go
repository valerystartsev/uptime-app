package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/config"
)

type fakeAuthenticator struct {
	registerResult auth.SessionResult
	registerErr    error
	loginErr       error
	updateResult   auth.Profile
	updateErr      error
	updatedName    *string
}

func (f fakeAuthenticator) Profile(context.Context, string) (auth.Profile, error) {
	return auth.Profile{}, nil
}

func (f fakeAuthenticator) UpdateProfile(_ context.Context, _, name string) (auth.Profile, error) {
	if f.updatedName != nil {
		*f.updatedName = name
	}
	return f.updateResult, f.updateErr
}

func (f fakeAuthenticator) Register(context.Context, string, string) (auth.SessionResult, error) {
	return f.registerResult, f.registerErr
}

func (f fakeAuthenticator) Login(context.Context, string, string) (auth.SessionResult, error) {
	return f.registerResult, f.loginErr
}

func (f fakeAuthenticator) Refresh(context.Context, string) (auth.SessionResult, error) {
	return f.registerResult, nil
}

func (f fakeAuthenticator) Logout(context.Context, string) error {
	return nil
}

func TestRegisterSetsRefreshCookie(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	handler := New(fakeAuthenticator{registerResult: auth.SessionResult{
		AccessToken:       "access-token",
		AccessTokenExpiry: time.Now().Add(time.Minute),
		RefreshToken:      "refresh-token",
		RefreshExpiresAt:  expiresAt,
	}}, config.CookieConfig{Name: "refresh_token", Path: "/", SameSite: http.SameSiteLaxMode})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password1"}`))
	recorder := httptest.NewRecorder()
	handler.Register(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "refresh_token" || !cookies[0].HttpOnly {
		t.Fatalf("cookies = %#v", cookies)
	}
	var response tokenResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AccessToken != "access-token" || response.TokenType != "Bearer" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRegisterMapsValidationError(t *testing.T) {
	handler := New(fakeAuthenticator{registerErr: auth.ErrWeakPassword}, config.CookieConfig{Name: "refresh_token", Path: "/"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"user@example.com","password":"short"}`))
	recorder := httptest.NewRecorder()
	handler.Register(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginMapsInvalidCredentials(t *testing.T) {
	handler := New(fakeAuthenticator{loginErr: auth.ErrInvalidCredentials}, config.CookieConfig{Name: "refresh_token", Path: "/"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password1"}`))
	recorder := httptest.NewRecorder()
	handler.Login(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestRegisterRejectsUnknownFields(t *testing.T) {
	handler := New(fakeAuthenticator{}, config.CookieConfig{Name: "refresh_token", Path: "/"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password1","role":"admin"}`))
	recorder := httptest.NewRecorder()
	handler.Register(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandleAuthErrorDoesNotExposeRefreshFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	handleAuthError(recorder, auth.ErrInvalidRefreshToken, false)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if errors.Is(auth.ErrInvalidRefreshToken, auth.ErrInvalidCredentials) {
		t.Fatal("test setup requires distinct errors")
	}
}

func TestUpdateProfileReturnsName(t *testing.T) {
	handler := New(fakeAuthenticator{updateResult: auth.Profile{Email: "user@example.com", Name: "Ada"}}, config.CookieConfig{})
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", strings.NewReader(`{"name":"Ada"}`))
	request.Header.Set("Authorization", "Bearer access-token")
	recorder := httptest.NewRecorder()
	handler.UpdateProfile(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var response profileResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Name != "Ada" || response.Email != "user@example.com" {
		t.Fatalf("response = %#v", response)
	}
}

func TestUpdateProfileSavesNameWithSpace(t *testing.T) {
	var updatedName string
	handler := New(fakeAuthenticator{
		updateResult: auth.Profile{Email: "user@example.com", Name: "Ada Lovelace"},
		updatedName:  &updatedName,
	}, config.CookieConfig{})
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", strings.NewReader(`{"name":"Ada Lovelace"}`))
	request.Header.Set("Authorization", "Bearer access-token")
	recorder := httptest.NewRecorder()
	handler.UpdateProfile(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if updatedName != "Ada Lovelace" {
		t.Fatalf("UpdateProfile() name = %q, want %q", updatedName, "Ada Lovelace")
	}
}

func TestUpdateProfileRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		token   string
		authErr error
		status  int
	}{
		{name: "missing token", body: `{"name":"Ada"}`, status: http.StatusUnauthorized},
		{name: "unknown field", body: `{"name":"Ada","role":"admin"}`, token: "Bearer token", status: http.StatusBadRequest},
		{name: "invalid name", body: `{"name":" "}`, token: "Bearer token", authErr: auth.ErrInvalidName, status: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := New(fakeAuthenticator{updateErr: tc.authErr}, config.CookieConfig{})
			request := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", strings.NewReader(tc.body))
			request.Header.Set("Authorization", tc.token)
			recorder := httptest.NewRecorder()
			handler.UpdateProfile(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
		})
	}
}
