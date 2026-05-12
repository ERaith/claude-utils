// pr-watcher is a Go daemon that polls GitHub + GitLab for open PRs/MRs,
// dispatches Claude Code reviews via the `claude` CLI, and fans review events
// out over a WebSocket. It never reads ANTHROPIC_API_KEY and never imports an
// Anthropic SDK; see docs/no-api-key-policy.md at the repo root.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/config"
	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/poller"
	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/reviewer"
	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/state"
	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/ws"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "pr-watcher: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "", "Path to YAML config (omit to use defaults + env vars)")
	flag.Parse()

	log := newLogger()
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Info("config loaded",
		"http_addr", cfg.HTTPAddr,
		"poll_interval", cfg.PollInterval.String(),
		"workers", cfg.ReviewWorkers,
		"daily_cap", cfg.DailyReviewCap,
		"state_file", cfg.StateFile,
		"repos", len(cfg.EnabledRepos()),
	)

	store, err := state.New(cfg.StateFile)
	if err != nil {
		return fmt.Errorf("state: %w", err)
	}

	hub := ws.New(log.With("component", "ws"))
	bridge := eventBridge{hub: hub}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	reviewCh := make(chan poller.ReviewRequest, 64)
	workerWG := reviewer.Pool(ctx, cfg.ReviewWorkers, reviewer.Config{
		ClaudeBin:      cfg.ClaudeBin,
		ReviewerAgent:  cfg.ReviewerAgent,
		ReviewerModel:  cfg.ReviewerModel,
		DailyReviewCap: cfg.DailyReviewCap,
		Store:          store,
		Events:         bridge,
		Log:            log.With("component", "reviewer"),
	}, reviewCh)

	// Pollers
	pollers := make([]*poller.Poller, 0, len(cfg.EnabledRepos()))
	var pollerWG sync.WaitGroup
	for _, r := range cfg.EnabledRepos() {
		p := &poller.Poller{
			Provider: r.Provider,
			Repo:     r.Repo,
			Interval: cfg.PollInterval,
			Store:    store,
			Out:      reviewCh,
			Events:   bridge,
			Log:      log.With("component", "poller"),
		}
		pollers = append(pollers, p)
		pollerWG.Add(1)
		go func(p *poller.Poller) {
			defer pollerWG.Done()
			log.Info("poller started", "provider", p.Provider, "repo", p.Repo)
			p.Run(ctx)
			log.Info("poller exited", "provider", p.Provider, "repo", p.Repo)
		}(p)
	}

	// HTTP / WS server
	mux := http.NewServeMux()
	mux.Handle("/ws", hub)
	mux.HandleFunc("/health", healthHandler(hub, store, pollers))
	mux.HandleFunc("/state", stateHandler(store))
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Info("http listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err.Error())
			cancel()
		}
	}()

	// Periodic state save
	saveDone := make(chan struct{})
	go func() {
		defer close(saveDone)
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := store.Save(); err != nil {
					log.Warn("state save failed", "err", err.Error())
				}
			}
		}
	}()

	// Wait for signal
	sig := <-sigCh
	log.Info("signal received, shutting down", "sig", sig.String())
	cancel()

	// Give pollers + workers up to 10s
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	done := make(chan struct{})
	go func() {
		pollerWG.Wait()
		close(reviewCh)
		workerWG.Wait()
		<-saveDone
		close(done)
	}()
	select {
	case <-done:
		log.Info("workers + pollers drained")
	case <-shutdownCtx.Done():
		log.Warn("shutdown timeout — forcing exit")
	}
	_ = server.Shutdown(shutdownCtx)
	hub.Shutdown()
	if err := store.Save(); err != nil {
		log.Warn("final state save failed", "err", err.Error())
	}
	log.Info("bye")
	return nil
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}

// eventBridge adapts the ws.Hub to the per-package EventSink interfaces. The
// poller and reviewer packages each define a small EventSink interface so they
// remain testable without depending on the websocket transport.
type eventBridge struct {
	hub *ws.Hub
}

func (b eventBridge) Publish(evt poller.Event) {
	b.hub.Publish(ws.Event{
		Type:      evt.Type,
		Provider:  evt.Provider,
		Repo:      evt.Repo,
		Number:    evt.Number,
		HeadSHA:   evt.HeadSHA,
		Title:     evt.Title,
		URL:       evt.URL,
		Timestamp: evt.Timestamp,
		Error:     evt.Error,
	})
}

type healthStats struct {
	OK              bool              `json:"ok"`
	Clients         int               `json:"connected_clients"`
	ReviewsToday    int               `json:"reviews_today"`
	StateFile       string            `json:"state_file"`
	Repos           []healthRepoEntry `json:"repos"`
	Uptime          string            `json:"uptime"`
	StartedAt       time.Time         `json:"started_at"`
	GoroutineEvents []string          `json:"-"`
}

type healthRepoEntry struct {
	Provider     string    `json:"provider"`
	Repo         string    `json:"repo"`
	LastPolledAt time.Time `json:"last_polled_at,omitempty"`
}

var startedAt = time.Now()

func healthHandler(hub *ws.Hub, store *state.Store, pollers []*poller.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repos := make([]healthRepoEntry, 0, len(pollers))
		for _, p := range pollers {
			repos = append(repos, healthRepoEntry{
				Provider:     p.Provider,
				Repo:         p.Repo,
				LastPolledAt: p.LastAt(),
			})
		}
		stats := healthStats{
			OK:           true,
			Clients:      hub.ClientCount(),
			ReviewsToday: store.ReviewsToday("", ""),
			StateFile:    store.Path(),
			Repos:        repos,
			Uptime:       time.Since(startedAt).Round(time.Second).String(),
			StartedAt:    startedAt,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stats)
	}
}

func stateHandler(store *state.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(store.Snapshot())
	}
}
