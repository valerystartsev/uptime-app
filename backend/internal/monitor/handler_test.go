package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/store"
)

type fakeManager struct {
	created store.Monitor
	items   []store.Monitor
	hasMore bool
	page    int
	err     error
}

func (f *fakeManager) Create(_ context.Context, token, url string, value int64, unit string) (store.Monitor, error) {
	if token != "token" {
		return store.Monitor{}, auth.ErrInvalidAccessToken
	}
	f.created = store.Monitor{ID: uuid.New(), URL: url, IntervalValue: value, IntervalUnit: unit, CreatedAt: time.Now()}
	return f.created, f.err
}

func (f *fakeManager) List(_ context.Context, token string, page int) ([]store.Monitor, bool, error) {
	if token != "token" {
		return nil, false, auth.ErrInvalidAccessToken
	}
	f.page = page
	return f.items, f.hasMore, f.err
}

func TestCreateMonitorReturnsSelectedInterval(t *testing.T) {
	manager := &fakeManager{}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(`{"url":"https://example.com","interval_value":7,"interval_unit":"seconds"}`))
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	NewHandler(manager).Create(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	var response monitorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.URL != "https://example.com" || response.IntervalValue != 7 || response.IntervalUnit != "seconds" || response.ID == "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestCreateMonitorRejectsBadRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body, token string
		managerErr        error
		status            int
	}{
		{"missing token", `{}`, "", nil, http.StatusUnauthorized},
		{"unknown field", `{"url":"https://example.com","interval_value":7,"interval_unit":"seconds","extra":1}`, "Bearer token", nil, http.StatusBadRequest},
		{"decimal interval", `{"url":"https://example.com","interval_value":1.5,"interval_unit":"seconds"}`, "Bearer token", nil, http.StatusBadRequest},
		{"invalid interval", `{"url":"https://example.com","interval_value":0,"interval_unit":"seconds"}`, "Bearer token", ErrInvalidInterval, http.StatusBadRequest},
		{"monitor limit", `{"url":"https://example.com","interval_value":1,"interval_unit":"seconds"}`, "Bearer token", ErrMonitorLimit, http.StatusConflict},
		{"invalid access token", `{"url":"https://example.com","interval_value":1,"interval_unit":"hours"}`, "Bearer invalid", nil, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(tc.body))
			request.Header.Set("Authorization", tc.token)
			recorder := httptest.NewRecorder()
			NewHandler(&fakeManager{err: tc.managerErr}).Create(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
		})
	}
}

func TestListMonitorsReturnsPage(t *testing.T) {
	manager := &fakeManager{items: []store.Monitor{{ID: uuid.New(), URL: "https://example.com", IntervalValue: 2, IntervalUnit: "hours", CreatedAt: time.Now()}}, hasMore: true}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors?page=2", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	NewHandler(manager).List(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var response struct {
		Items   []monitorResponse `json:"items"`
		HasMore bool              `json:"has_more"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if manager.page != 2 || !response.HasMore || len(response.Items) != 1 || response.Items[0].IntervalValue != 2 || response.Items[0].IntervalUnit != "hours" {
		t.Fatalf("response = %#v", response)
	}
}

func TestListMonitorsRejectsInvalidPage(t *testing.T) {
	for _, page := range []string{"0", "3", "abc"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors?page="+page, nil)
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		NewHandler(&fakeManager{}).List(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("page %q: status = %d, want 400", page, recorder.Code)
		}
	}
}
