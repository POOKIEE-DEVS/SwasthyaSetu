// Command server runs the SwasthyaSetu backend: the API, the WebSockets and
// the built frontend, from one process and one origin.
//
// Run exactly one instance. The patient queue and the call rooms live in
// memory, so a second instance would split them and calls would fail.
//
//	server               serve on $PORT (default 8000)
//	server -healthcheck  exit 0 if the local server answers /health
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ai"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/api"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check the running server's /health and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(1)
	}
	if *healthcheck {
		os.Exit(checkHealth(cfg.Port))
	}
	log := logging.New(os.Stdout, cfg.LogLevel, cfg.LogJSON)
	if err := run(cfg, log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Never fatal: the emergency chat and patient calls work without it.
	db := database.Open(cfg.DatabaseURL, log)
	defer db.Close()
	databaseReady := db.TryInit(ctx)

	model := ai.NewMedGemma(cfg.HFSpaceID, cfg.HFToken, cfg.ModelTimeout, log)
	app := api.New(cfg, db, model, log)
	defer app.Close()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		// No read or write timeout: calls hold WebSockets open for as long
		// as they last, and a chat reply can take minutes on a cold GPU.
	}

	log.Info("backend starting",
		"version", api.Version,
		"environment", cfg.Environment,
		"port", cfg.Port,
		"model_configured", cfg.ModelConfigured(),
		"turn_configured", cfg.TURNConfigured(),
		"database", db.Kind(),
		"database_ready", databaseReady,
		"google_sign_in", cfg.GoogleConfigured(),
		"serving_frontend", app.ServingFrontend(),
	)

	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	app.Shutdown() // tell call sockets to reconnect elsewhere
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// checkHealth is the container's health check: the image has no shell or
// curl, so the binary checks itself.
func checkHealth(port string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "unhealthy:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "unhealthy:", resp.Status)
		return 1
	}
	return 0
}
