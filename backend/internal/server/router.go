package server

import (
	"github.com/google/uuid"
	"net/http"
	"path"
	"strings"

	"example.com/uptime/backend/internal/auth/controller"
	"example.com/uptime/backend/internal/config"
	"example.com/uptime/backend/internal/files"
	"example.com/uptime/backend/internal/monitor"
)

func NewRouter(authenticator controller.Authenticator, monitors monitor.Manager, cfg config.Config) http.Handler {
	handler := controller.New(authenticator, cfg.RefreshCookie)
	monitorHandler := monitor.NewHandler(monitors)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", handler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", handler.Logout)
	mux.HandleFunc("GET /api/v1/auth/me", handler.Me)
	mux.HandleFunc("PATCH /api/v1/auth/me", handler.UpdateProfile)
	mux.HandleFunc("POST /api/v1/auth/me/avatar", handler.UploadAvatar)
	mux.HandleFunc("POST /api/v1/monitors", monitorHandler.Create)
	mux.HandleFunc("GET /api/v1/monitors", monitorHandler.List)
	mux.HandleFunc("GET /api/v1/avatars/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := r.PathValue("filename")
		id, ext := strings.TrimSuffix(strings.TrimSuffix(filename, ".png"), ".jpg"), path.Ext(filename)
		if _, err := uuid.Parse(id); err != nil || (ext != ".png" && ext != ".jpg") || filename != id+ext {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		path, err := files.New(cfg.FilesDir).Path("avatars", filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})
	return cors(cfg.CORSAllowedOrigin, mux)
}

func cors(allowedOrigin string, next http.Handler) http.Handler {
	if allowedOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
