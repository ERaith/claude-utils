#!/usr/bin/env bash
# session-start.template.sh
#
# Claude Code SessionStart hook. Stdout is injected into Claude's context
# at session start, so it knows project state immediately.
#
# Install via settings.json:
#   {
#     "hooks": {
#       "SessionStart": [{
#         "matcher": "startup",
#         "hooks": [{"type": "command", "command": "/path/to/session-start.sh"}]
#       }]
#     }
#   }
#
# Or use the installer:
#   bash ~/claude-utils/hooks/claude-code/install-session-start.sh
#
# Customize the CUSTOMIZE blocks below for your project.

set -uo pipefail

# ── Universal context (cheap, always useful) ──────────────────────────────
HOST=$(hostname -s)
CWD=$(pwd | sed "s|$HOME|~|")
DATE=$(date '+%Y-%m-%d %H:%M')

# Git repo context (only if cwd is inside a git repo).
GIT_CONTEXT=""
if git rev-parse --git-dir >/dev/null 2>&1; then
  BRANCH=$(git branch --show-current 2>/dev/null || echo "(detached)")
  AHEAD_BEHIND=$(git rev-list --left-right --count HEAD...@{upstream} 2>/dev/null | awk '{print $1" ahead, "$2" behind"}')
  DIRTY=$(git status --porcelain 2>/dev/null | wc -l | xargs)
  GIT_CONTEXT="- **Branch**: ${BRANCH} | ${AHEAD_BEHIND:-no upstream} | dirty: ${DIRTY}"
fi

# ── Project CLAUDE.md hint (universal) ────────────────────────────────────
CLAUDE_MD_HINT=""
if [[ -f CLAUDE.md ]]; then
  CLAUDE_MD_HINT="Read CLAUDE.md for project conventions and todos."
fi

# ── CUSTOMIZE: project-specific context ───────────────────────────────────
# Add anything cheap and useful: storage, container health, recent activity.
# Keep total output under ~50 lines. Examples below — uncomment / adapt.
PROJECT_CONTEXT=""
#
# # Disk usage (homelab example):
# DISK=$(df -h . | awk 'NR==2{print $5" of "$2}')
# PROJECT_CONTEXT+="
# - **Disk**: ${DISK}"
#
# # Docker (homelab example):
# UNHEALTHY=$(docker ps --filter health=unhealthy --format '{{.Names}}' 2>/dev/null | tr '\n' ', ' | sed 's/,$//')
# PROJECT_CONTEXT+="
# - **Unhealthy containers**: ${UNHEALTHY:-none}"
#
# # Recent CI status (work repo example):
# CI=$(gh run list -L1 --json conclusion -q '.[0].conclusion' 2>/dev/null)
# PROJECT_CONTEXT+="
# - **Last CI run**: ${CI:-unknown}"

# ── CUSTOMIZE: memory graph context (optional) ────────────────────────────
# If you've installed the relationship-memory graph from claude-utils, pull
# top-N relevant nodes here. Falls back silently if not installed.
MEMORY_CONTEXT=""
if command -v memory_graph.py >/dev/null 2>&1; then
  MEMORY_CONTEXT=$(memory_graph.py context 2>/dev/null | grep -v -E "Warning|Loading|BertModel|UNEXPECTED|Notes" || true)
elif [[ -x "$HOME/.claude/memory_graph.py" ]]; then
  MEMORY_CONTEXT=$("$HOME/.claude/memory_graph.py" context 2>/dev/null | grep -v -E "Warning|Loading|BertModel|UNEXPECTED|Notes" || true)
fi

# ── Emit the context block ─────────────────────────────────────────────────
cat <<EOF
## Session Context
- **Host**: ${HOST} | **Time**: ${DATE}
- **CWD**: ${CWD}
${GIT_CONTEXT}${PROJECT_CONTEXT}
EOF

if [[ -n "$CLAUDE_MD_HINT" ]]; then
  echo ""
  echo "$CLAUDE_MD_HINT"
fi

if [[ -n "$MEMORY_CONTEXT" ]]; then
  echo ""
  echo "$MEMORY_CONTEXT"
fi
