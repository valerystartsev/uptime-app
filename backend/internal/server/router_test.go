package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/uptime/backend/internal/config"
	"example.com/uptime/backend/internal/files"
)

func TestAvatarIsServedFromSharedFileRoot(t *testing.T) {
	root := t.TempDir()
	name, err := files.New(root).Save("avatars", ".png", []byte("avatar"))
	if err != nil {
		t.Fatal(err)
	}
	router := NewRouter(nil, config.Config{FilesDir: root})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+name, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "avatar" {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
}

func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := cors("http://localhost:3000", next)
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/refresh", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}

	blocked := httptest.NewRecorder()
	blockedRequest := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/refresh", nil)
	blockedRequest.Header.Set("Origin", "https://attacker.example")
	handler.ServeHTTP(blocked, blockedRequest)
	if got := blocked.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}
