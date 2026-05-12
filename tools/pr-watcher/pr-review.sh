#!/usr/bin/env bash
# pr-review.sh — fallback shell wrapper for reviewing a single PR/MR without
# running the Go daemon. The Go binary is the supported path; this script
# exists for emergencies, debugging, or hosts where Go is not installed.
#
# Usage:
#   pr-review.sh github  owner/name  42
#   pr-review.sh gitlab  group/proj  17  [--dry-run]
#
# Auth is inherited from `gh auth login` / `glab auth login`. Claude is invoked
# via the local `claude` CLI — no API keys, ever. See docs/no-api-key-policy.md.

set -euo pipefail

if [[ $# -lt 3 ]]; then
    echo "usage: $0 <github|gitlab> <repo> <number> [--dry-run]" >&2
    exit 2
fi

PROVIDER="$1"
REPO="$2"
NUMBER="$3"
DRY_RUN="${4:-}"

REVIEWER_AGENT="${REVIEWER_AGENT:-code-reviewer}"
CLAUDE_BIN="${CLAUDE_BIN:-claude}"

case "$PROVIDER" in
    github)
        DIFF="$(gh pr diff "$NUMBER" --repo "$REPO")"
        TITLE="$(gh pr view "$NUMBER" --repo "$REPO" --json title -q .title)"
        URL="$(gh pr view "$NUMBER" --repo "$REPO" --json url -q .url)"
        HEAD_SHA="$(gh pr view "$NUMBER" --repo "$REPO" --json headRefOid -q .headRefOid)"
        ;;
    gitlab)
        DIFF="$(glab mr diff "$NUMBER" -R "$REPO")"
        TITLE="$(glab mr view "$NUMBER" -R "$REPO" -F json | jq -r .title)"
        URL="$(glab mr view "$NUMBER" -R "$REPO" -F json | jq -r .web_url)"
        HEAD_SHA="$(glab mr view "$NUMBER" -R "$REPO" -F json | jq -r '.diff_refs.head_sha // .sha')"
        ;;
    *)
        echo "unknown provider: $PROVIDER" >&2
        exit 2
        ;;
esac

if [[ -z "$DIFF" ]]; then
    echo "empty diff — nothing to review" >&2
    exit 0
fi

MARKER="<!-- pr-watcher:${HEAD_SHA} -->"

PROMPT=$(cat <<EOF
You are reviewing a pull request. Provide a concise, actionable review:
- Call out correctness, security, and regression risks.
- Note testability or documentation gaps when relevant.
- Do not approve or request changes via the platform — comment only.
- Keep the response in well-formatted Markdown.

Repository: ${REPO}
PR/MR number: ${NUMBER}
Title: ${TITLE}
URL: ${URL}
Head SHA: ${HEAD_SHA}

Unified diff follows:

\`\`\`diff
${DIFF}
\`\`\`
EOF
)

REVIEW="$(printf '%s\n' "$PROMPT" | "$CLAUDE_BIN" -p --agent "$REVIEWER_AGENT")"
BODY="${REVIEW}

${MARKER}
"

if [[ "$DRY_RUN" == "--dry-run" ]]; then
    printf '%s\n' "$BODY"
    exit 0
fi

case "$PROVIDER" in
    github) gh pr comment "$NUMBER" --repo "$REPO" --body "$BODY" ;;
    gitlab) glab mr note "$NUMBER" -R "$REPO" -m "$BODY" ;;
esac
