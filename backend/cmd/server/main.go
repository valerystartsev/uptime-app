package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/uptime/backend/internal/auth"
	"example.com/uptime/backend/internal/config"
	"example.com/uptime/backend/internal/files"
	"example.com/uptime/backend/internal/migrations"
	"example.com/uptime/backend/internal/server"
	"example.com/uptime/backend/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dataStore, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := dataStore.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := migrations.Run(context.Background(), dataStore.DB()); err != nil {
		log.Fatal(err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	authService := auth.NewService(dataStore, tokens, cfg.RefreshTokenTTL)
	authService.SetFileStorage(files.New(cfg.FilesDir))
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.NewRouter(authService, cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
