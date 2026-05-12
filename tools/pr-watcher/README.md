# pr-watcher

A single-binary Go daemon that polls one or more GitHub / GitLab repos for
open PRs/MRs, dispatches a Claude Code review for each new head SHA, posts the
review back as a PR comment, and fans the lifecycle out over a websocket so a
web UI can subscribe.

## Why

- **No API keys, no SDK.** Calls Claude exclusively via the `claude` CLI, which
  uses your Claude **Pro** or **Max** subscription. The daemon never reads
  `ANTHROPIC_API_KEY` and never imports an Anthropic SDK. See
  [`docs/no-api-key-policy.md`](../../docs/no-api-key-policy.md) at the repo
  root for the full policy.
- **Lighter than webhooks.** No GitHub App, no GitLab runner, no public
  endpoint. Poll on a cron-like interval from a host you already own.
- **Stdlib-first.** Two third-party deps total: a websocket library and a YAML
  parser. The binary is small, the dependency surface is small.

## Install

```bash
go install github.com/ERaith/claude-utils/tools/pr-watcher@latest
```

Prerequisites on the host:

- `gh` and/or `glab` installed and authenticated (`gh auth login`,
  `glab auth login`). The daemon inherits their token storage — it never
  touches credentials itself.
- `claude` CLI installed and logged in to your Pro/Max subscription.

## Configuration

Copy `config.example.yaml` to `~/.config/pr-watcher/config.yaml` and edit:

```yaml
poll_interval: 60s          # min 5s
review_workers: 2           # concurrent claude invocations
daily_review_cap: 20        # per repo, rolling 24h
state_file: ~/.cache/pr-watcher/state.json
http_addr: ":8782"
claude_bin: claude          # PATH lookup by default
reviewer_agent: code-reviewer
# reviewer_model: sonnet    # optional; forwarded to `claude --model`

repos:
  - provider: github
    repo: ERaith/claude-utils
    enabled: true
  - provider: gitlab
    repo: my-group/my-project
    enabled: false
```

Every field is also overridable via env var. The env value wins over YAML.

| Env var                       | Type       | Notes                       |
| ----------------------------- | ---------- | --------------------------- |
| `PR_WATCHER_POLL_INTERVAL`    | `duration` | e.g. `60s`, `2m`            |
| `PR_WATCHER_REVIEW_WORKERS`   | `int`      |                             |
| `PR_WATCHER_DAILY_REVIEW_CAP` | `int`      |                             |
| `PR_WATCHER_STATE_FILE`       | `path`     | `~` expanded                |
| `PR_WATCHER_HTTP_ADDR`        | `string`   | e.g. `:8782` or `127.0.0.1:8782` |
| `PR_WATCHER_CLAUDE_BIN`       | `string`   |                             |
| `PR_WATCHER_REVIEWER_AGENT`   | `string`   |                             |
| `PR_WATCHER_REVIEWER_MODEL`   | `string`   |                             |
| `LOG_LEVEL`                   | `string`   | `debug`/`info`/`warn`/`error` |

## Run

```bash
pr-watcher --config ~/.config/pr-watcher/config.yaml
```

Or rely entirely on defaults + env vars (no `--config` flag).

## HTTP endpoints

- `GET /ws` — WebSocket. Streams every lifecycle event as JSON.
- `GET /health` — JSON status: connected clients, reviews in last 24h, per-repo
  `last_polled_at`, uptime.
- `GET /state` — Raw dedup state snapshot.

## WebSocket event schema

Every message is a single JSON object on its own frame.

| Field        | Type   | Notes                                                                |
| ------------ | ------ | -------------------------------------------------------------------- |
| `type`       | string | `pr_opened`, `review_started`, `review_posted`, `review_failed`      |
| `provider`   | string | `github` or `gitlab`                                                 |
| `repo`       | string | `owner/name`                                                         |
| `number`     | int    | PR/MR number                                                         |
| `head_sha`   | string |                                                                      |
| `title`      | string |                                                                      |
| `url`        | string | For `review_posted` this is the comment URL                          |
| `timestamp`  | RFC3339 |                                                                      |
| `error`      | string | Only present when `type=review_failed`                               |

Example:

```json
{"type":"review_posted","provider":"github","repo":"ERaith/claude-utils","number":42,"head_sha":"abc123def","title":"feat: pr-watcher","url":"https://github.com/ERaith/claude-utils/pull/42#issuecomment-...","timestamp":"2026-05-12T11:00:00Z"}
```

## systemd --user install

Drop the unit file in place:

```bash
mkdir -p ~/.config/systemd/user
cp systemd/pr-watcher.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now pr-watcher.service
journalctl --user -u pr-watcher.service -f
```

For the daemon to keep running across logouts:

```bash
sudo loginctl enable-linger "$USER"
```

## Behavior

- Polls each enabled repo every `poll_interval` (per-repo goroutine).
- Dedup key: `<provider>:<repo>:<number>:<head_sha>`. New keys are queued for
  review.
- Each review posts a comment ending with the HTML marker
  `<!-- pr-watcher:<head_sha> -->`. On the next poll, if the marker is already
  present in the PR comments, the head SHA is skipped — safe across restarts
  even if state is lost.
- Hard cap of `daily_review_cap` reviews per repo per rolling 24h window.
- State persists to JSON every 30s and on graceful shutdown (SIGTERM/SIGINT,
  10s drain).

## Security notes

- **Tokens stay in `gh`/`glab` keyrings.** The daemon shells out to those CLIs
  and inherits whatever auth they hold. It never reads `GH_TOKEN`,
  `GITHUB_TOKEN`, `GITLAB_TOKEN` itself.
- **No code execution.** The reviewer fetches the unified diff via
  `gh pr diff` / `glab mr diff` and pipes that text to `claude`. It never
  checks out PR branches or runs PR code.
- **Run as a non-privileged user.** A dedicated UNIX user with only access to
  its own state file and `gh`/`glab`/`claude` binaries is recommended. The
  shipped systemd unit hardens with `NoNewPrivileges`, `PrivateTmp`, and
  `ProtectSystem=strict`.
- **No Anthropic SDK, no API key.** The hard policy is documented at
  [`docs/no-api-key-policy.md`](../../docs/no-api-key-policy.md). A pre-commit
  grep enforces it; this tool intentionally has zero references to the API.

## Reference

See the team-agent definition this tool dispatches at
[`agents/team/pr-watcher.md`](../../agents/team/pr-watcher.md). That doc
describes the contract for the reviewer subagent — `pr-watcher` (this binary)
is the polling/orchestration layer, the agent file is the prompt contract.

## Development

```bash
cd tools/pr-watcher
go vet ./...
go test ./...
go build ./...
```

Tests cover the load-bearing logic: state dedup, poller URL building, and the
websocket hub's non-blocking broadcast behavior. Subprocess calls (`gh`,
`glab`, `claude`) are not unit-tested — they're verified by running the binary
against a real repo.
