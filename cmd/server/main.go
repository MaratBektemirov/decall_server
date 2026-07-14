package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"decall_server/internal/auth"
	"decall_server/internal/callid"
	"decall_server/internal/config"
	"decall_server/internal/middleware"
	signaling "decall_server/internal/signal"
	"decall_server/internal/turn"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	authHandler := auth.NewHandler(cfg)
	turnHandler := turn.NewHandler(cfg, authHandler)
	signalHub := signaling.NewHub()
	signalHandler := signaling.NewHandler(cfg, signalHub)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /auth/challenge", middleware.WithCORS(cfg.CORSOrigins, authHandler.IssueChallenge))
	mux.HandleFunc("OPTIONS /auth/challenge", middleware.WithCORS(cfg.CORSOrigins, func(w http.ResponseWriter, r *http.Request) {}))

	// For generate Call ID
	mux.HandleFunc("GET /generate-id", middleware.WithCORS(cfg.CORSOrigins, func(w http.ResponseWriter, r *http.Request) {
		pubKey := r.URL.Query().Get("pubkey")
		if pubKey == "" {
			http.Error(w, `{"error": "pubkey is required"}`, http.StatusBadRequest)
			return
		}

		callID := callid.GenerateID(pubKey)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": callID})
	}))

	// Allowing pre requests for the browser
	mux.HandleFunc("OPTIONS /generate-id", middleware.WithCORS(cfg.CORSOrigins, func(w http.ResponseWriter, r *http.Request) {}))

	mux.HandleFunc("POST /turn-credentials", middleware.WithCORS(cfg.CORSOrigins, turnHandler.IssueCredentials))
	mux.HandleFunc("OPTIONS /turn-credentials", middleware.WithCORS(cfg.CORSOrigins, func(w http.ResponseWriter, r *http.Request) {}))

	mux.Handle("GET /signal", signalHandler)

	addr := ":8080"
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
