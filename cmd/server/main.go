package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LuizArtur29/http-server-projeto-korp/internal/handler"
	"github.com/LuizArtur29/http-server-projeto-korp/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)

	slog.SetDefault(logger)

	metrics := middleware.NewMetrics()

	mux := http.NewServeMux()

	mux.Handle(
		"/projeto-korp",
		metrics.Observe(
			"/projeto-korp",
			http.HandlerFunc(handler.ProjetoKorp),
		),
	)

	mux.Handle(
		"/healthz",
		metrics.Observe(
			"/healthz",
			http.HandlerFunc(handler.Health),
		),
	)

	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info(
			"starting HTTP server",
			"address", server.Addr,
		)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			slog.Error(
				"Http server failer",
				"error", err,
			)
			os.Exit(1)

		}
	}()

	shutdown := make(chan os.Signal, 1)

	signal.Notify(
		shutdown,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-shutdown

	slog.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error(
			"graceful shutdown failed",
			"error", err,
		)

		os.Exit(1)
	}

	slog.Info("server stopped")
}
