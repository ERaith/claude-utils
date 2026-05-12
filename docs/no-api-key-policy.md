# No-API-Key Policy

## The rule

**This repo's agents call Claude exclusively via the `claude` CLI.** No code in this repo, and no code that consumes these agents, may:

- Read or reference the environment variable `ANTHROPIC_API_KEY`.
- Import or depend on the `anthropic` Python package.
- Import or depend on the `@anthropic-ai/sdk` TypeScript / JavaScript package.
- Call `https://api.anthropic.com/` directly (curl, fetch, requests, etc.).
- Set `Authorization: Bearer sk-ant-...` on any HTTP request.

The only sanctioned way to invoke Claude is:

```bash
claude -p --model <model> --agent <path-to-agent.md> < <input>
```

## Why

The consumers of this repo (a homelab automation user, a separate work agent) are on Claude **Pro** and **Max** subscriptions. Those are flat-rate plans. Calling the Anthropic API directly bills **per-token** against a separate billing path — the subscription does not cover API calls. A single accidental import of the `anthropic` SDK can quietly rack up real-money charges on a separate invoice while the subscription sits idle.

This is not a stylistic preference. It is a cost-control invariant. Mixing the two billing paths defeats the entire reason for a subscription.

## Enforcement

### grep patterns that catch violations

Run these in CI or as a pre-commit hook. Any hit is a fail:

```bash
# Python SDK
grep -RIn -E '^\s*(import|from)\s+anthropic\b' --include='*.py' .

# JS / TS SDK
grep -RIn -E "from\s+['\"]@anthropic-ai/sdk['\"]" --include='*.ts' --include='*.tsx' --include='*.js' --include='*.mjs' .
grep -RIn -E "require\(\s*['\"]@anthropic-ai/sdk['\"]\s*\)" --include='*.ts' --include='*.js' .

# API key references in code
grep -RIn -E '\bANTHROPIC_API_KEY\b' --include='*.py' --include='*.ts' --include='*.js' --include='*.sh' --include='*.md' .

# Direct API host
grep -RIn -E 'api\.anthropic\.com' --include='*.py' --include='*.ts' --include='*.js' --include='*.sh' .

# Bearer token shape
grep -RIn -E 'Bearer\s+sk-ant-' .
```

A passing repo returns zero matches across all six commands. (`.md` files referencing the policy by name are fine — those will only match the literal `ANTHROPIC_API_KEY` token inside this document, which is explicit guidance, not a code path. If you want a stricter rule, exclude `docs/no-api-key-policy.md` and the agent `forbidden:` lines from the grep.)

### Pre-commit hook (drop into `.git/hooks/pre-commit`)

```bash
#!/usr/bin/env bash
set -euo pipefail

VIOLATIONS=$(
  {
    grep -RIn -E '^\s*(import|from)\s+anthropic\b' --include='*.py' . 2>/dev/null
    grep -RIn -E "from\s+['\"]@anthropic-ai/sdk['\"]" --include='*.ts' --include='*.tsx' --include='*.js' --include='*.mjs' . 2>/dev/null
    grep -RIn -E 'api\.anthropic\.com' --include='*.py' --include='*.ts' --include='*.js' --include='*.sh' . 2>/dev/null
    grep -RIn -E 'Bearer\s+sk-ant-' . 2>/dev/null
  } | grep -v '^docs/no-api-key-policy.md' | grep -v '^agents/team/.*\.md' || true
)

if [ -n "$VIOLATIONS" ]; then
  echo "no-api-key-policy violation:" >&2
  echo "$VIOLATIONS" >&2
  echo "" >&2
  echo "See docs/no-api-key-policy.md. Use 'claude -p' instead." >&2
  exit 1
fi
```

### Agent-level enforcement

Every agent definition in `agents/team/` carries a `forbidden:` list in its frontmatter:

```yaml
forbidden: [ANTHROPIC_API_KEY, anthropic-python, anthropic-typescript]
```

The downstream loader (your wrapper script, the homelab Forge, the work agent) reads this list and refuses to dispatch the agent if any of those tokens appear in the input task or in the spec the agent is consuming. This is belt-and-suspenders on top of the grep check.

## If you really need the API

You don't. Whatever you're trying to do, the `claude` CLI can do it:

- Need a one-shot response -> `claude -p "prompt"`.
- Need streaming -> the CLI streams by default.
- Need to pin a model -> `claude -p --model opus "prompt"`.
- Need to chain calls -> wrap `claude -p` invocations in a bash script and parse the output.
- Need structured output -> have the agent emit a `*_START..END` fenced block and grep it.

If after reading this you still believe a use case requires the API, raise it with a human before adding any SDK import. The default answer is no.

## Why this lives in a `docs/` file, not just a code comment

So that anyone vendoring these agents into a third repo can read the rule, understand the cost reason, and ship the grep patterns into their own CI. The policy travels with the agents.
