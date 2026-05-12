# Hooks

Two layers of automation:

- **`git/`** — Git hooks. Run during `git commit` / `git push`. Block bad commits before they happen.
- **`claude-code/`** — Claude Code hooks. Run on session lifecycle events (start, stop, pre-tool-use). Inject context, log activity, gate dangerous actions.

## Quick install (per repo)

From inside a target git repo:

```bash
# Git hooks
bash ~/claude-utils/hooks/git/install.sh

# Claude Code session-start hook (writes to ~/.claude/settings.json)
bash ~/claude-utils/hooks/claude-code/install-session-start.sh
```

Both installers are idempotent. They symlink rather than copy so `git pull` in claude-utils updates them automatically.

## What's here

| Hook | Layer | Purpose |
|------|-------|---------|
| `git/pre-commit-no-api-key.sh` | git | Enforces the no-API-key policy. Blocks commits containing `ANTHROPIC_API_KEY`, `sk-ant-...`, `import anthropic`, `@anthropic-ai/sdk`. |
| `claude-code/session-start.template.sh` | Claude Code | Template that injects project context at session start. Customize per repo. |

## The no-API-key policy

Subscribers (Claude Pro/Max) pay a flat fee that includes Claude Code usage. Calling the Anthropic API directly is **per-token billing** — it bypasses the subscription and creates a surprise bill. See [`docs/no-api-key-policy.md`](../docs/no-api-key-policy.md) for the full rule.

The git hook is the enforcement point. Anything that gets past it (e.g. cherry-picked from another branch, `--no-verify` bypass) is caught by CI checks if you wire those up.

## Bypassing

You shouldn't need to. But if you're committing documentation that legitimately discusses the API:

- File matches `docs/no-api-key-policy*` → auto-allowed
- File matches `*.example` or `*.template` → auto-allowed
- Line contains `# noqa: anthropic-api` or `// noqa: anthropic-api` → auto-allowed
- Last resort: `git commit --no-verify` (this leaves a trail — don't use it for code)
