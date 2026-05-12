// Package reviewer dispatches code-reviewer subagent calls via the `claude` CLI
// and posts the result back as a PR/MR comment.
//
// IMPORTANT: This package shells out to the `claude` CLI exclusively. It does
// not import any Anthropic SDK and never reads ANTHROPIC_API_KEY. See
// docs/no-api-key-policy.md at the repo root.
package reviewer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/poller"
	"github.com/ERaith/claude-utils/tools/pr-watcher/internal/state"
)

// Config carries everything a worker needs.
type Config struct {
	ClaudeBin      string
	ReviewerAgent  string // value passed to `claude --agent`
	ReviewerModel  string // optional; passed to `claude --model`
	DailyReviewCap int
	Store          *state.Store
	Events         EventSink
	Log            *slog.Logger
}

// EventSink is the minimal interface used to broadcast reviewer events.
type EventSink interface {
	Publish(evt poller.Event)
}

// Worker reads ReviewRequests off In and processes them.
type Worker struct {
	ID  int
	Cfg Config
	In  <-chan poller.ReviewRequest
}

// Pool spins up N workers and returns a WaitGroup that signals when all exit.
func Pool(ctx context.Context, n int, cfg Config, in <-chan poller.ReviewRequest) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		w := &Worker{ID: i, Cfg: cfg, In: in}
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Run(ctx)
		}()
	}
	return &wg
}

// Run pulls requests until ctx is done or In is closed.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-w.In:
			if !ok {
				return
			}
			w.handle(ctx, req)
		}
	}
}

func (w *Worker) handle(ctx context.Context, req poller.ReviewRequest) {
	pr := req.PR
	key := pr.DedupKey()
	log := w.Cfg.Log.With("worker", w.ID, "provider", pr.Provider, "repo", pr.Repo, "number", pr.Number, "head_sha", pr.HeadSHA)

	// Daily cap check (per-repo).
	if got := w.Cfg.Store.ReviewsToday(pr.Provider, pr.Repo); got >= w.Cfg.DailyReviewCap {
		log.Warn("daily review cap reached, skipping", "cap", w.Cfg.DailyReviewCap, "count", got)
		w.Cfg.Store.Update(key, func(e state.Entry) state.Entry {
			e.Status = "skipped_cap"
			return e
		})
		return
	}

	// Idempotency: if marker exists in comments, skip.
	marker := fmt.Sprintf("<!-- pr-watcher:%s -->", pr.HeadSHA)
	if existing, err := poller.FetchExistingComments(ctx, pr.Provider, pr.Repo, pr.Number); err == nil {
		if strings.Contains(existing, marker) {
			log.Info("marker already present, skipping")
			w.Cfg.Store.Update(key, func(e state.Entry) state.Entry {
				e.Status = "already_reviewed"
				e.LastReviewed = time.Now().UTC()
				return e
			})
			return
		}
	} else {
		// Non-fatal: we still attempt the review.
		log.Debug("could not fetch existing comments", "err", err.Error())
	}

	w.publish(poller.Event{
		Type: "review_started", Provider: pr.Provider, Repo: pr.Repo,
		Number: pr.Number, HeadSHA: pr.HeadSHA, Title: pr.Title, URL: pr.URL,
		Timestamp: time.Now().UTC(),
	})
	w.Cfg.Store.Update(key, func(e state.Entry) state.Entry {
		e.Status = "reviewing"
		return e
	})

	diff, err := poller.FetchDiff(ctx, pr.Provider, pr.Repo, pr.Number)
	if err != nil {
		w.fail(key, pr, fmt.Errorf("fetch diff: %w", err), log)
		return
	}
	if strings.TrimSpace(diff) == "" {
		w.fail(key, pr, fmt.Errorf("empty diff"), log)
		return
	}

	review, err := w.runClaude(ctx, pr, diff)
	if err != nil {
		w.fail(key, pr, fmt.Errorf("claude: %w", err), log)
		return
	}
	body := strings.TrimSpace(review) + "\n\n" + marker + "\n"

	url, err := poller.PostComment(ctx, pr.Provider, pr.Repo, pr.Number, body)
	if err != nil {
		w.fail(key, pr, fmt.Errorf("post comment: %w", err), log)
		return
	}
	now := time.Now().UTC()
	w.Cfg.Store.Update(key, func(e state.Entry) state.Entry {
		e.Status = "reviewed"
		e.LastReviewed = now
		e.ReviewURL = url
		e.LastError = ""
		return e
	})
	log.Info("review posted", "url", url)
	w.publish(poller.Event{
		Type: "review_posted", Provider: pr.Provider, Repo: pr.Repo,
		Number: pr.Number, HeadSHA: pr.HeadSHA, Title: pr.Title, URL: url,
		Timestamp: now,
	})
}

func (w *Worker) fail(key string, pr poller.PR, err error, log *slog.Logger) {
	log.Error("review failed", "err", err.Error())
	w.Cfg.Store.Update(key, func(e state.Entry) state.Entry {
		e.Status = "failed"
		e.LastError = err.Error()
		return e
	})
	w.publish(poller.Event{
		Type: "review_failed", Provider: pr.Provider, Repo: pr.Repo,
		Number: pr.Number, HeadSHA: pr.HeadSHA, Title: pr.Title, URL: pr.URL,
		Error: err.Error(), Timestamp: time.Now().UTC(),
	})
}

func (w *Worker) publish(evt poller.Event) {
	if w.Cfg.Events != nil {
		w.Cfg.Events.Publish(evt)
	}
}

// runClaude invokes the `claude` CLI in print mode against the configured agent
// and returns its stdout (the review body).
func (w *Worker) runClaude(ctx context.Context, pr poller.PR, diff string) (string, error) {
	prompt := buildPrompt(pr, diff)
	args := []string{"-p", "--agent", w.Cfg.ReviewerAgent}
	if w.Cfg.ReviewerModel != "" {
		args = append(args, "--model", w.Cfg.ReviewerModel)
	}
	cmd := exec.CommandContext(ctx, w.Cfg.ClaudeBin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", fmt.Errorf("claude produced empty review (stderr: %s)", strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func buildPrompt(pr poller.PR, diff string) string {
	var b strings.Builder
	b.WriteString("You are reviewing a pull request. Provide a concise, actionable review:\n")
	b.WriteString("- Call out correctness, security, and regression risks.\n")
	b.WriteString("- Note testability or documentation gaps when relevant.\n")
	b.WriteString("- Do not approve or request changes via the platform — comment only.\n")
	b.WriteString("- Keep the response in well-formatted Markdown.\n\n")
	fmt.Fprintf(&b, "Repository: %s\n", pr.Repo)
	fmt.Fprintf(&b, "PR/MR number: %d\n", pr.Number)
	fmt.Fprintf(&b, "Title: %s\n", pr.Title)
	fmt.Fprintf(&b, "URL: %s\n", pr.URL)
	fmt.Fprintf(&b, "Head SHA: %s\n\n", pr.HeadSHA)
	b.WriteString("Unified diff follows:\n\n")
	b.WriteString("```diff\n")
	b.WriteString(diff)
	if !strings.HasSuffix(diff, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n")
	return b.String()
}
