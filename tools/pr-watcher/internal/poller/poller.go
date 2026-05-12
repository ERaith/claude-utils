// Package poller polls a single repo (GitHub via `gh` or GitLab via `glab`) and
// emits ReviewRequest values for previously-unseen head_sha observations.
package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/state"
)

// PR is the normalized record across providers.
type PR struct {
	Provider  string
	Repo      string
	Number    int
	HeadSHA   string
	Title     string
	URL       string
	UpdatedAt time.Time
}

// DedupKey returns the canonical state key for this PR.
func (p PR) DedupKey() string { return state.Key(p.Provider, p.Repo, p.Number, p.HeadSHA) }

// ReviewRequest is what the poller pushes onto the worker channel.
type ReviewRequest struct {
	PR PR
}

// Poller polls a single repo until ctx is canceled.
type Poller struct {
	Provider string
	Repo     string
	Interval time.Duration
	Store    *state.Store
	Out      chan<- ReviewRequest
	Events   EventSink
	Log      *slog.Logger
	// LastPolledAt is updated on each poll for /health.
	lastAt time.Time
}

// EventSink is the minimal interface the poller uses to broadcast events.
type EventSink interface {
	Publish(evt Event)
}

// Event is what flows to the websocket hub.
type Event struct {
	Type      string    `json:"type"`
	Repo      string    `json:"repo"`
	Provider  string    `json:"provider,omitempty"`
	Number    int       `json:"number,omitempty"`
	HeadSHA   string    `json:"head_sha,omitempty"`
	Title     string    `json:"title,omitempty"`
	URL       string    `json:"url,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// LastAt returns the most recent successful poll timestamp for /health stats.
func (p *Poller) LastAt() time.Time { return p.lastAt }

// Run loops until ctx is canceled.
func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.Interval)
	defer t.Stop()
	// Tick immediately on start.
	p.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	prs, err := p.list(ctx)
	if err != nil {
		p.Log.Warn("poll failed",
			"provider", p.Provider, "repo", p.Repo, "err", err.Error())
		return
	}
	p.lastAt = time.Now().UTC()
	for _, pr := range prs {
		entry := state.Entry{
			Provider: pr.Provider,
			Repo:     pr.Repo,
			Number:   pr.Number,
			HeadSHA:  pr.HeadSHA,
			Title:    pr.Title,
			URL:      pr.URL,
		}
		isNew := p.Store.MarkSeen(entry)
		if !isNew {
			continue
		}
		if p.Events != nil {
			p.Events.Publish(Event{
				Type:      "pr_opened",
				Provider:  pr.Provider,
				Repo:      pr.Repo,
				Number:    pr.Number,
				HeadSHA:   pr.HeadSHA,
				Title:     pr.Title,
				URL:       pr.URL,
				Timestamp: time.Now().UTC(),
			})
		}
		select {
		case p.Out <- ReviewRequest{PR: pr}:
		case <-ctx.Done():
			return
		default:
			p.Log.Warn("review queue full, dropping",
				"provider", pr.Provider, "repo", pr.Repo, "number", pr.Number)
		}
	}
}

func (p *Poller) list(ctx context.Context) ([]PR, error) {
	switch p.Provider {
	case "github":
		return ListGitHub(ctx, p.Repo)
	case "gitlab":
		return ListGitLab(ctx, p.Repo)
	default:
		return nil, fmt.Errorf("unknown provider %q", p.Provider)
	}
}

// runner is overridable in tests; in production it execs the named binary.
var runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// ghCLI / glabCLI are the names of the binaries; overridable in tests.
var (
	ghCLI   = "gh"
	glabCLI = "glab"
)

// BuildGitHubArgs returns the args we pass to `gh pr list`. Exposed for tests.
func BuildGitHubArgs(repo string) []string {
	return []string{
		"pr", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,headRefOid,updatedAt,title,url",
		"--limit", "100",
	}
}

// BuildGitLabArgs returns the args we pass to `glab mr list`. Exposed for tests.
func BuildGitLabArgs(repo string) []string {
	return []string{
		"mr", "list",
		"-R", repo,
		"--opened",
		"-F", "json",
		"--per-page", "100",
	}
}

// ListGitHub shells out to `gh` and returns normalized PRs.
func ListGitHub(ctx context.Context, repo string) ([]PR, error) {
	out, err := runner(ctx, ghCLI, BuildGitHubArgs(repo)...)
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	var raw []struct {
		Number     int       `json:"number"`
		HeadRefOID string    `json:"headRefOid"`
		UpdatedAt  time.Time `json:"updatedAt"`
		Title      string    `json:"title"`
		URL        string    `json:"url"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse gh json: %w", err)
	}
	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		if r.HeadRefOID == "" {
			continue
		}
		prs = append(prs, PR{
			Provider:  "github",
			Repo:      repo,
			Number:    r.Number,
			HeadSHA:   r.HeadRefOID,
			Title:     r.Title,
			URL:       r.URL,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return prs, nil
}

// ListGitLab shells out to `glab` and returns normalized MRs.
//
// glab field names vary slightly across versions; we accept a few common shapes.
func ListGitLab(ctx context.Context, repo string) ([]PR, error) {
	out, err := runner(ctx, glabCLI, BuildGitLabArgs(repo)...)
	if err != nil {
		return nil, fmt.Errorf("glab mr list: %w", err)
	}
	var raw []map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse glab json: %w", err)
	}
	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		number := intField(r, "iid", "number")
		if number == 0 {
			continue
		}
		headSHA := strField(r, "sha", "head_sha", "headSHA", "headRefOid")
		if headSHA == "" {
			// Some glab versions nest the SHA under "diff_refs.head_sha".
			if dr, ok := r["diff_refs"].(map[string]any); ok {
				headSHA = strField(dr, "head_sha")
			}
		}
		if headSHA == "" {
			continue
		}
		title := strField(r, "title")
		url := strField(r, "web_url", "url")
		updated := parseTime(strField(r, "updated_at", "updatedAt"))
		prs = append(prs, PR{
			Provider:  "gitlab",
			Repo:      repo,
			Number:    number,
			HeadSHA:   headSHA,
			Title:     title,
			URL:       url,
			UpdatedAt: updated,
		})
	}
	return prs, nil
}

func intField(m map[string]any, keys ...string) int {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			var x int
			fmt.Sscanf(n, "%d", &x)
			if x != 0 {
				return x
			}
		}
	}
	return 0
}

func strField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// PostComment posts the review body as a PR/MR comment via the appropriate CLI.
// It returns the URL of the new comment, when available.
func PostComment(ctx context.Context, provider, repo string, number int, body string) (string, error) {
	switch provider {
	case "github":
		cmd := exec.CommandContext(ctx, ghCLI, "pr", "comment",
			fmt.Sprint(number), "--repo", repo, "--body", body)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("gh pr comment: %w: %s", err, strings.TrimSpace(string(out)))
		}
		// gh prints the URL of the created comment on stdout.
		return strings.TrimSpace(string(out)), nil
	case "gitlab":
		cmd := exec.CommandContext(ctx, glabCLI, "mr", "note",
			fmt.Sprint(number), "-R", repo, "-m", body)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("glab mr note: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", fmt.Errorf("unknown provider %q", provider)
	}
}

// FetchDiff returns the unified diff for a PR/MR.
func FetchDiff(ctx context.Context, provider, repo string, number int) (string, error) {
	switch provider {
	case "github":
		cmd := exec.CommandContext(ctx, ghCLI, "pr", "diff",
			fmt.Sprint(number), "--repo", repo)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("gh pr diff: %w", err)
		}
		return string(out), nil
	case "gitlab":
		cmd := exec.CommandContext(ctx, glabCLI, "mr", "diff",
			fmt.Sprint(number), "-R", repo)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("glab mr diff: %w", err)
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unknown provider %q", provider)
	}
}

// FetchExistingComments returns the raw text of existing PR/MR comments so the
// reviewer can skip if the marker is already present.
func FetchExistingComments(ctx context.Context, provider, repo string, number int) (string, error) {
	switch provider {
	case "github":
		cmd := exec.CommandContext(ctx, ghCLI, "pr", "view",
			fmt.Sprint(number), "--repo", repo, "--comments")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("gh pr view: %w", err)
		}
		return string(out), nil
	case "gitlab":
		// glab does not have a single command for fetching all notes; fall back
		// to `mr view` which prints recent notes.
		cmd := exec.CommandContext(ctx, glabCLI, "mr", "view",
			fmt.Sprint(number), "-R", repo, "--comments")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("glab mr view: %w", err)
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unknown provider %q", provider)
	}
}
