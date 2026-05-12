# Claude Code Hooks

Hooks that run on Claude Code session lifecycle events. Customize per project.

## Available hooks

| Hook | Event | Purpose |
|------|-------|---------|
| `session-start.template.sh` | `SessionStart` | Injects project context (host, cwd, git status, memory graph) at session start |

## Install

```bash
bash ~/claude-utils/hooks/claude-code/install-session-start.sh
```

This copies the template to `~/.claude/hooks/session-start.sh` and registers it in `~/.claude/settings.json`. Won't overwrite an existing customized hook.

## Customizing

Open `~/.claude/hooks/session-start.sh` and look for `CUSTOMIZE` blocks. Common additions:

- **Storage**: `df -h` on your data drive
- **Container health**: `docker ps --filter health=unhealthy`
- **Recent CI**: `gh run list -L1`
- **Active downloads / queues**: hit your *arr / Transmission / sonarr API
- **Memory graph**: top-N relevant nodes from a sqlite-vec store (see `memory-graph/` once it lands)

Keep stdout under ~50 lines. The context window is precious.

## Other event types you can hook

Claude Code supports more than `SessionStart`. See [the docs](https://code.claude.com/docs/hooks) for the full list. Useful ones:

- `PreToolUse` (matcher: `Write|Edit`) — gate dangerous writes (e.g. block API-key references — belt-and-suspenders with the git pre-commit)
- `PostToolUse` — log activity, run formatters
- `Stop` — write session summary to memory graph, post to Discord
- `UserPromptSubmit` — preprocess user input (e.g. expand custom shortcuts)
