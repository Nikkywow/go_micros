package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"go-microservice/handlers"
	"go-microservice/metrics"
	"go-microservice/services"
	"go-microservice/utils"
)

func main() {
	logger := utils.NewLogger()

	integration, err := services.NewIntegrationServiceFromEnv(context.Background())
	if err != nil {
		log.Fatalf("integration init failed: %v", err)
	}

	audit := services.NewAuditService(logger, integration)
	defer audit.Close()

	userService := services.NewUserService()
	userHandler := handlers.NewUserHandler(userService, audit)
	integrationHandler := handlers.NewIntegrationHandler(integration)

	router := mux.NewRouter()
	router.Use(utils.RecoverMiddleware(logger))
	router.Use(utils.NewRateLimitMiddleware(rate.Limit(1000), 5000))
	router.Use(metrics.Middleware)

	api := router.PathPrefix("/api").Subrouter()
	userHandler.Register(api)
	integrationHandler.Register(api)

	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet)

	addr := env("APP_PORT", ":8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	waitForShutdown(srv, logger)
}

func waitForShutdown(srv *http.Server, logger interface{ Info(string, ...any); Error(string, ...any) }) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return
	}
	logger.Info("server stopped")
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
