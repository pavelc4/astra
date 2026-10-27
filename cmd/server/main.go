package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"github.com/pavelc4/astra/internal/config"
	"github.com/pavelc4/astra/internal/handler"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		TimeFormat: time.StampMilli,
	})))

	slog.Info("cookies check",
		"instagram_loaded", os.Getenv("INSTAGRAM_COOKIES") != "",
		"facebook_loaded", os.Getenv("FACEBOOK_COOKIES") != "",
	)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	handler.New().Register(r)

	addr := cfg.Host + ":" + cfg.Port

	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	// SIGHUP listener: reload cookies from .env on the fly without stopping server (pkill -HUP astra-server)
	hupSig := make(chan os.Signal, 1)
	signal.Notify(hupSig, syscall.SIGHUP)
	go func() {
		for range hupSig {
			slog.Info("received SIGHUP: reloading cookies from .env...")
			reloaded, err := config.ReloadCookies()
			if err != nil {
				slog.Error("failed to reload cookies from .env", "error", err)
			} else {
				slog.Info("cookies reloaded successfully without restart", "platforms", reloaded)
			}
		}
	}()

	go func() {
		slog.Info("server starting", "host", cfg.Host, "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-stopSig
	slog.Info("server stopping gracefully", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server graceful shutdown failed", "error", err)
		if err := server.Close(); err != nil {
			slog.Error("forced server close failed", "error", err)
		}
	}

	slog.Info("server exited cleanly")
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handler.TotalRequests, 1)

		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		if ww.Status() >= 400 {
			atomic.AddUint64(&handler.FailedRequests, 1)
		} else {
			atomic.AddUint64(&handler.SuccessRequests, 1)
		}

		slog.LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
