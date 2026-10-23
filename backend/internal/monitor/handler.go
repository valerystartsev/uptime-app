package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/store"
)

type Manager interface {
	Create(context.Context, string, string, int64, string) (store.Monitor, error)
	List(context.Context, string) ([]store.Monitor, error)
}

type Handler struct{ manager Manager }

func NewHandler(manager Manager) *Handler { return &Handler{manager: manager} }

type createRequest struct {
	URL           string `json:"url"`
	IntervalValue int64  `json:"interval_value"`
	IntervalUnit  string `json:"interval_unit"`
}

type monitorResponse struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`
	IntervalValue int64     `json:"interval_value"`
	IntervalUnit  string    `json:"interval_unit"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	token, ok := accessToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body createRequest
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	monitor, err := h.manager.Create(r.Context(), token, body.URL, body.IntervalValue, body.IntervalUnit)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(monitor))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	token, ok := accessToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid access token")
		return
	}
	monitors, err := h.manager.List(r.Context(), token)
	if err != nil {
		handleError(w, err)
		return
	}
	result := make([]monitorResponse, 0, len(monitors))
	for _, monitor := range monitors {
		result = append(result, toResponse(monitor))
	}
	writeJSON(w, http.StatusOK, result)
}

func accessToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	return token, found && scheme == "Bearer" && token != ""
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidURL), errors.Is(err, ErrInvalidInterval):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrInvalidAccessToken):
		writeError(w, http.StatusUnauthorized, "invalid access token")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func toResponse(monitor store.Monitor) monitorResponse {
	return monitorResponse{ID: monitor.ID.String(), URL: monitor.URL,
		IntervalValue: monitor.IntervalValue, IntervalUnit: monitor.IntervalUnit, CreatedAt: monitor.CreatedAt}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}
