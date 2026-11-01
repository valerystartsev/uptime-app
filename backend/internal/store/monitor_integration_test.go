package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/uptime/backend/internal/migrations"
	"example.com/uptime/backend/internal/store"
)

func TestMonitorLimitAndPages(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	dataStore, err := store.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dataStore.Close() })
	user := store.User{ID: uuid.New(), Email: uuid.NewString() + "@example.com", PasswordHash: "test", CreatedAt: time.Now().UTC()}
	if err := dataStore.CreateUser(ctx, &user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Delete(&user) })
	for i := 0; i < 100; i++ {
		monitor := store.Monitor{ID: uuid.New(), UserID: user.ID, URL: fmt.Sprintf("https://example.com/%d", i),
			IntervalValue: 1, IntervalUnit: "minutes", CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Microsecond)}
		if err := dataStore.CreateMonitor(ctx, &monitor); err != nil {
			t.Fatalf("create monitor %d: %v", i, err)
		}
	}
	monitor := store.Monitor{ID: uuid.New(), UserID: user.ID, URL: "https://example.com/extra", IntervalValue: 1, IntervalUnit: "minutes", CreatedAt: time.Now().UTC()}
	if err := dataStore.CreateMonitor(ctx, &monitor); !errors.Is(err, store.ErrMonitorLimit) {
		t.Fatalf("101st monitor: got %v, want ErrMonitorLimit", err)
	}
	// Older accounts could have exceeded the new limit.
	for i := 0; i < 21; i++ {
		legacy := store.Monitor{ID: uuid.New(), UserID: user.ID, URL: fmt.Sprintf("https://example.com/legacy/%d", i), IntervalValue: 1, IntervalUnit: "minutes", CreatedAt: time.Now().UTC().Add(-time.Hour)}
		if err := db.Create(&legacy).Error; err != nil {
			t.Fatal(err)
		}
	}
	seen := map[uuid.UUID]bool{}
	var cursor *store.MonitorCursor
	for page := 0; ; page++ {
		items, err := dataStore.ListMonitors(ctx, user.ID, 50, cursor)
		if err != nil {
			t.Fatal(err)
		}
		hasMore := len(items) > 50
		if hasMore {
			items = items[:50]
		}
		for _, item := range items {
			if seen[item.ID] {
				t.Fatalf("duplicate monitor on page %d: %s", page+1, item.ID)
			}
			seen[item.ID] = true
		}
		if page == 0 {
			newer := store.Monitor{ID: uuid.New(), UserID: user.ID, URL: "https://example.com/newer", IntervalValue: 1, IntervalUnit: "minutes", CreatedAt: time.Now().UTC().Add(time.Hour)}
			if err := db.Create(&newer).Error; err != nil {
				t.Fatal(err)
			}
		}
		if !hasMore {
			break
		}
		last := items[len(items)-1]
		cursor = &store.MonitorCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if len(seen) != 121 {
		t.Fatalf("got %d existing monitors, want 121", len(seen))
	}
}

func TestMonitorDuplicateAndConcurrentCreation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	dataStore, err := store.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dataStore.Close() })
	if err := migrations.Run(ctx, dataStore.DB()); err != nil {
		t.Fatal(err)
	}
	users := []store.User{
		{ID: uuid.New(), Email: uuid.NewString() + "@example.com", PasswordHash: "test", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), Email: uuid.NewString() + "@example.com", PasswordHash: "test", CreatedAt: time.Now().UTC()},
	}
	for _, user := range users {
		if err := dataStore.CreateUser(ctx, &user); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { dataStore.DB().Delete(&user) })
	}
	newMonitor := func(userID uuid.UUID, address string) *store.Monitor {
		return &store.Monitor{ID: uuid.New(), UserID: userID, URL: address, IntervalValue: 1, IntervalUnit: "minutes", CreatedAt: time.Now().UTC()}
	}
	// Insert a legacy URL directly to verify that preexisting records are considered.
	if err := dataStore.DB().Create(newMonitor(users[0].ID, "https://EXAMPLE.com:443")).Error; err != nil {
		t.Fatal(err)
	}
	if err := dataStore.CreateMonitor(ctx, newMonitor(users[0].ID, "https://example.com/")); !errors.Is(err, store.ErrMonitorExists) {
		t.Fatalf("duplicate legacy URL: got %v", err)
	}
	if err := dataStore.CreateMonitor(ctx, newMonitor(users[1].ID, "https://example.com/")); err != nil {
		t.Fatalf("same URL for another user: %v", err)
	}
	if err := dataStore.CreateMonitor(ctx, newMonitor(users[0].ID, "https://example.com/another")); err != nil {
		t.Fatalf("different path: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, address := range []string{"https://example.com/concurrent", "https://EXAMPLE.COM:443/concurrent"} {
		workers.Add(1)
		go func(address string) {
			defer workers.Done()
			<-start
			results <- dataStore.CreateMonitor(ctx, newMonitor(users[0].ID, address))
		}(address)
	}
	close(start)
	workers.Wait()
	close(results)
	success, duplicates := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, store.ErrMonitorExists):
			duplicates++
		default:
			t.Fatalf("concurrent create: %v", err)
		}
	}
	if success != 1 || duplicates != 1 {
		t.Fatalf("concurrent creation: successes = %d, duplicates = %d", success, duplicates)
	}
}
