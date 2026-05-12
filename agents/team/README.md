# Team Agents

A coordinated set of Claude Code agents designed to work together: route a task, investigate it, build it, verify it, fix it on rejection, and run regression sweeps over time. Plus three utility agents for news digests, PR watching, and weekly self-improvement.

All agents run via the `claude` CLI on your subscription — **never** the Anthropic API. See `docs/no-api-key-policy.md` for why and how that is enforced.

## The team

| Agent | Model | Tools | Purpose |
|---|---|---|---|
| `router` | haiku | Read, Grep | Pick which agent handles a task |
| `researcher` | opus | Read, Grep, Glob, WebSearch, WebFetch | Investigate, produce a spec |
| `builder` | opus | Read, Write, Edit, Bash, Grep, Glob, WebSearch | Implement the spec |
| `reviewer` | sonnet | Read, Grep, Bash | Verify the build (no write access) |
| `fixer` | opus | Read, Write, Edit, Bash, Grep, Glob, WebSearch | Second attempt with a different approach |
| `auditor` | sonnet | Read, Grep, Bash | Periodic regression sweep from a checks file |
| `evolution` | opus | Read, Write, Edit, Grep, Glob, Bash | Weekly meta-agent — proposes patches to the team |
| `news-brewer` | sonnet | WebSearch, WebFetch, Write, Read | Personalized daily digest |
| `pr-watcher` | sonnet | Bash, Read | Polls a repo for PRs, dispatches code-reviewer, posts comments |

## When to use which

- **New task, unclear destination** -> `router`
- **Symptom but no diagnosis** -> `researcher` -> `builder` -> `reviewer`
- **Clear spec, just do it** -> `builder` -> `reviewer`
- **Reviewer said `reject`** -> `fixer` -> `reviewer`
- **Reviewer said `needs-human`** -> stop, escalate
- **Nightly checks** -> `auditor`
- **Weekly self-tuning** -> `evolution`
- **Daily news brief** -> `news-brewer`
- **PRs piling up** -> `pr-watcher`

## Hand-off diagram

```
                +--------+
   new task --> | router |
                +---+----+
                    |
        +-----------+-----------+
        |                       |
        v                       v
  +-----------+           +---------+
  | researcher|---------->| builder |
  +-----------+   spec    +----+----+
                               |
                          build report
                               |
                               v
                         +----------+
                         | reviewer |
                         +----+-----+
                              |
              +---------------+----------------+
              |               |                |
            pass            reject       needs-human
              |               |                |
              v               v                v
            merge          +-------+         (stop,
                           | fixer |         escalate)
                           +---+---+
                               |
                               +--> reviewer (one more swing, then escalate)


  (in parallel, on cron)

  +---------+        +-----------+        +-------------+
  | auditor |        | evolution |        | news-brewer |
  +---------+        +-----------+        +-------------+
  nightly            weekly                daily

  +-------------+
  | pr-watcher  |   on interval
  +-------------+
```

## Output contracts

Every agent emits a single fenced block with `*_START` / `*_END` markers so wrapper scripts can grep for it without parsing free-form text:

- `router` -> `ROUTING_DECISION_START..END`
- `researcher` -> `RESEARCH_SPEC_START..END`
- `builder` / `fixer` -> `BUILD_REPORT_START..END`
- `reviewer` -> `REVIEW_RESULT_START..END`
- `auditor` -> `AUDIT_REPORT_START..END`
- `evolution` -> `EVOLUTION_REPORT_START..END`
- `news-brewer` -> `NEWS_BRIEF_START..END` + a written markdown file
- `pr-watcher` -> `PR_WATCH_REPORT_START..END`

## Install

### Option A — symlink into your global agents dir

```bash
mkdir -p ~/.claude/agents
ln -s "$(pwd)/agents/team" ~/.claude/agents/team
```

Now you can invoke any team agent from anywhere:

```bash
claude -p --agent ~/.claude/agents/team/builder.md < /tmp/spec.txt
```

### Option B — curl install

From a fresh machine:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/ERaith/claude-utils/master/setup.sh)
```

The repo's `setup.sh` already wires `~/claude-utils` into your environment; with this branch merged it will pick up `agents/team/` automatically.

### Option C — copy per-repo

If you want a different version of an agent in a specific repo, copy the relevant `.md` into that repo's `.claude/agents/team/` and tune it. The agent loader prefers repo-local definitions over global ones.

## Calling pattern (all agents)

```bash
claude -p --model <model> --agent <path-to-agent.md> <<'EOF'
<structured input matching the agent's "Input contract" section>
EOF
```

There is no Python wrapper, no `anthropic` SDK, no API key. The `claude` CLI uses your Pro/Max subscription. If you find yourself reaching for an SDK call, stop and read `docs/no-api-key-policy.md`.

## Wiring agents into a chain

A minimal end-to-end script:

```bash
#!/usr/bin/env bash
set -euo pipefail

TASK="$1"

# 1. Route
DECISION=$(claude -p --model haiku --agent ~/.claude/agents/team/router.md <<<"$TASK")

# 2. Parse the routing block, dispatch downstream...
# (your wrapper extracts agent: and context_to_load: and invokes the next claude -p)
```

For a fuller example see your project's `automation/forge/forge.sh` — that script demonstrates the BUILD_REPORT extraction pattern.
