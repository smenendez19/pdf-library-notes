package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdfnotes-backend/internal/config"
	"pdfnotes-backend/internal/documents"
	apihttp "pdfnotes-backend/internal/http"
	"pdfnotes-backend/internal/notes"
	"pdfnotes-backend/internal/store"
)

const version = "v0.1.0"

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","time":"` + time.Now().UTC().Format(time.RFC3339) + `","version":"` + version + `"}`))
}

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.StorageRoot, 0o755); err != nil {
		slog.Error("storage root error", "err", err)
		os.Exit(1)
	}
	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("database error", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	docs := documents.NewHandler(st, cfg.StorageRoot)
	docs.Register(mux)
	mux.HandleFunc("POST /api/v1/documents/upload", docs.Upload)
	nh := notes.NewHandler(st)
	nh.Register(mux)

	var handler http.Handler = mux
	handler = apihttp.Chain(handler, cfg.AllowedOrigins)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
