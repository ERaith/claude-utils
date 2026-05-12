// Package config loads pr-watcher YAML configuration with env-var overrides.
//
// Env overrides (set after YAML is parsed, so env wins):
//
//	PR_WATCHER_POLL_INTERVAL    e.g. "60s"
//	PR_WATCHER_REVIEW_WORKERS   integer
//	PR_WATCHER_DAILY_REVIEW_CAP integer
//	PR_WATCHER_STATE_FILE       path (~ is expanded)
//	PR_WATCHER_HTTP_ADDR        e.g. ":8782"
package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Repo describes a single repo to poll.
type Repo struct {
	Provider string `yaml:"provider"` // "github" or "gitlab"
	Repo     string `yaml:"repo"`     // e.g. "ERaith/claude-utils"
	Enabled  bool   `yaml:"enabled"`
}

// Config is the parsed pr-watcher configuration.
type Config struct {
	PollInterval   time.Duration `yaml:"poll_interval"`
	ReviewWorkers  int           `yaml:"review_workers"`
	DailyReviewCap int           `yaml:"daily_review_cap"`
	StateFile      string        `yaml:"state_file"`
	HTTPAddr       string        `yaml:"http_addr"`
	Repos          []Repo        `yaml:"repos"`
	// ClaudeBin is the path/name of the claude CLI (default "claude").
	ClaudeBin string `yaml:"claude_bin"`
	// ReviewerAgent is the path passed to `claude --agent`. Default "code-reviewer".
	ReviewerAgent string `yaml:"reviewer_agent"`
	// ReviewerModel is forwarded to `claude --model` if set.
	ReviewerModel string `yaml:"reviewer_model"`
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		PollInterval:   60 * time.Second,
		ReviewWorkers:  2,
		DailyReviewCap: 20,
		StateFile:      "~/.cache/pr-watcher/state.json",
		HTTPAddr:       ":8782",
		ClaudeBin:      "claude",
		ReviewerAgent:  "code-reviewer",
	}
}

// Load parses YAML from path (if non-empty), applies env overrides, and validates.
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config %q: %w", path, err)
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("parse yaml %q: %w", path, err)
		}
	}
	applyEnvOverrides(&cfg)
	expanded, err := ExpandTilde(cfg.StateFile)
	if err != nil {
		return cfg, err
	}
	cfg.StateFile = expanded
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Validate checks required fields and value ranges.
func (c Config) Validate() error {
	if c.PollInterval < 5*time.Second {
		return fmt.Errorf("poll_interval %s too small (min 5s)", c.PollInterval)
	}
	if c.ReviewWorkers < 1 {
		return errors.New("review_workers must be >= 1")
	}
	if c.DailyReviewCap < 1 {
		return errors.New("daily_review_cap must be >= 1")
	}
	if c.HTTPAddr == "" {
		return errors.New("http_addr must be set")
	}
	if c.StateFile == "" {
		return errors.New("state_file must be set")
	}
	if c.ClaudeBin == "" {
		return errors.New("claude_bin must be set")
	}
	enabled := 0
	for i, r := range c.Repos {
		if r.Provider != "github" && r.Provider != "gitlab" {
			return fmt.Errorf("repos[%d].provider %q: must be github or gitlab", i, r.Provider)
		}
		if r.Repo == "" && r.Enabled {
			return fmt.Errorf("repos[%d]: repo is empty but enabled=true", i)
		}
		if r.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return errors.New("at least one repo must be enabled")
	}
	return nil
}

// EnabledRepos returns the subset of repos with enabled=true and a non-empty name.
func (c Config) EnabledRepos() []Repo {
	out := make([]Repo, 0, len(c.Repos))
	for _, r := range c.Repos {
		if r.Enabled && r.Repo != "" {
			out = append(out, r)
		}
	}
	return out
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("PR_WATCHER_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.PollInterval = d
		}
	}
	if v := os.Getenv("PR_WATCHER_REVIEW_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.ReviewWorkers = n
		}
	}
	if v := os.Getenv("PR_WATCHER_DAILY_REVIEW_CAP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.DailyReviewCap = n
		}
	}
	if v := os.Getenv("PR_WATCHER_STATE_FILE"); v != "" {
		c.StateFile = v
	}
	if v := os.Getenv("PR_WATCHER_HTTP_ADDR"); v != "" {
		c.HTTPAddr = v
	}
	if v := os.Getenv("PR_WATCHER_CLAUDE_BIN"); v != "" {
		c.ClaudeBin = v
	}
	if v := os.Getenv("PR_WATCHER_REVIEWER_AGENT"); v != "" {
		c.ReviewerAgent = v
	}
	if v := os.Getenv("PR_WATCHER_REVIEWER_MODEL"); v != "" {
		c.ReviewerModel = v
	}
}

// ExpandTilde replaces a leading "~" or "~/" with the current user's home dir.
func ExpandTilde(p string) (string, error) {
	if p == "" || !strings.HasPrefix(p, "~") {
		return p, nil
	}
	if p == "~" {
		u, err := user.Current()
		if err != nil {
			return p, err
		}
		return u.HomeDir, nil
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p, err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
