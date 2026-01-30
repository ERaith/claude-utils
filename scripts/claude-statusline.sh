#!/bin/bash
# Claude Code status line script
# Shows: Model | Current Directory | Git Branch (if in repo)

# Get model from environment or default
MODEL="${CLAUDE_MODEL:-sonnet}"

# Get current directory (shortened)
CWD=$(pwd | sed "s|$HOME|~|")

# Get git info if in a repo
GIT_INFO=""
if git rev-parse --git-dir > /dev/null 2>&1; then
    BRANCH=$(git branch --show-current 2>/dev/null)
    if [ -n "$BRANCH" ]; then
        # Check if there are uncommitted changes
        if ! git diff-index --quiet HEAD -- 2>/dev/null; then
            GIT_INFO=" | ${BRANCH}*"
        else
            GIT_INFO=" | ${BRANCH}"
        fi
    fi
fi

# Output the status line
echo "${MODEL} | ${CWD}${GIT_INFO}"
