package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"yu-tasks/internal/config"
	"yu-tasks/internal/platform/db"
)

//go:embed web migrations
var assets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.Open(ctx, cfg.DBPath, assets)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           routes(webFS, cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("YU Tasks listening on %s (%s)", cfg.Address, cfg.Environment)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func routes(webFS fs.FS, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(securityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	r.Get("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if cfg.DevLogin {
			_, _ = w.Write([]byte(`{"devLogin":true,"languages":["ru","en"]}`))
			return
		}
		_, _ = w.Write([]byte(`{"devLogin":false,"languages":["ru","en"]}`))
	})
	r.Handle("/*", http.FileServer(http.FS(webFS)))
	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'", "img-src 'self' data:", "style-src 'self'",
			"script-src 'self'", "connect-src 'self'", "object-src 'none'",
			"base-uri 'self'", "frame-ancestors 'none'",
		}, "; "))
		next.ServeHTTP(w, r)
	})
}
