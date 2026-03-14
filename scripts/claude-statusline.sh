#!/bin/bash
# Claude Code status line — homelab edition
# Shows: model | host | cwd | git | storage | downloads

# Model
MODEL="${CLAUDE_MODEL:-sonnet}"

# Hostname — critical when SSH'd into multiple machines
HOST=$(hostname -s)

# Current directory (shortened)
CWD=$(pwd | sed "s|$HOME|~|")

# Git branch
GIT_INFO=""
if git rev-parse --git-dir > /dev/null 2>&1; then
    BRANCH=$(git branch --show-current 2>/dev/null)
    if [ -n "$BRANCH" ]; then
        if ! git diff-index --quiet HEAD -- 2>/dev/null; then
            GIT_INFO=" | ${BRANCH}*"
        else
            GIT_INFO=" | ${BRANCH}"
        fi
    fi
fi

# Storage: show bigboy usage (primary media drive) — fast df, no hang
STORAGE=""
BIGBOY=$(df /media/bigboy 2>/dev/null | awk 'NR==2{printf "%s", $5}')
if [ -n "$BIGBOY" ]; then
    STORAGE=" | bigboy:${BIGBOY}"
fi

# Active Transmission downloads (fast RPC call, 1s timeout)
DL_INFO=""
ACTIVE=$(curl -s --max-time 1 -u "admin:H1bWjCblJyboRqhQAuQxlKa/" \
    -H "X-Transmission-Session-Id: $(curl -s --max-time 1 -u 'admin:H1bWjCblJyboRqhQAuQxlKa/' \
        http://10.100.0.2:9091/transmission/rpc 2>/dev/null | grep -o 'X-Transmission-Session-Id: [^<]*' | cut -d' ' -f2)" \
    -d '{"method":"torrent-get","arguments":{"fields":["status"]}}' \
    http://10.100.0.2:9091/transmission/rpc 2>/dev/null | \
    python3 -c "import json,sys; d=json.load(sys.stdin); t=d.get('arguments',{}).get('torrents',[]); print(sum(1 for x in t if x.get('status')==4))" 2>/dev/null)
if [ -n "$ACTIVE" ] && [ "$ACTIVE" != "0" ]; then
    DL_INFO=" | dl:${ACTIVE}"
fi

# Output
echo "${MODEL} | ${HOST} | ${CWD}${GIT_INFO}${STORAGE}${DL_INFO}"
