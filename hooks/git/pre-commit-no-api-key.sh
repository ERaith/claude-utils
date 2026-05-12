#!/usr/bin/env bash
# pre-commit-no-api-key.sh
# Blocks commits that violate the no-API-key policy (see docs/no-api-key-policy.md).
#
# Install (from a target repo):
#   ln -sf "$HOME/claude-utils/hooks/git/pre-commit-no-api-key.sh" \
#          "$(git rev-parse --git-dir)/hooks/pre-commit"
#
# Or use the installer:
#   bash ~/claude-utils/hooks/git/install.sh

set -euo pipefail

# Patterns that indicate API-key usage (not allowed).
# Each pattern is a POSIX extended regex evaluated against ADDED lines only.
PATTERNS=(
  'ANTHROPIC_API_KEY'
  'sk-ant-[A-Za-z0-9_-]{20,}'
  '^[[:space:]]*import[[:space:]]+anthropic([[:space:]]|$|\.)'
  '^[[:space:]]*from[[:space:]]+anthropic[[:space:]]+import'
  '@anthropic-ai/sdk'
  'require\(["'\'']@anthropic-ai/sdk["'\'']\)'
)

# Files where the patterns above are legitimate (e.g. the policy doc itself).
ALLOWLIST_PATHS=(
  'docs/no-api-key-policy'
  '\.example$'
  '\.template$'
  'hooks/git/pre-commit-no-api-key\.sh$'   # this file
)

# Inline opt-out marker for individual lines.
NOQA_MARKER='noqa: anthropic-api'

is_allowed_path() {
  local path="$1"
  for allow in "${ALLOWLIST_PATHS[@]}"; do
    if echo "$path" | grep -qE "$allow"; then
      return 0
    fi
  done
  return 1
}

# Collect staged file paths (added/modified only).
mapfile -t STAGED_FILES < <(git diff --cached --name-only --diff-filter=AM)
[[ ${#STAGED_FILES[@]} -eq 0 ]] && exit 0

VIOLATIONS=()

for file in "${STAGED_FILES[@]}"; do
  if is_allowed_path "$file"; then
    continue
  fi

  # Get the added lines for this file, skip diff headers.
  # Strip the leading '+' so patterns can anchor on real source-code starts.
  added=$(git diff --cached --diff-filter=AM -U0 -- "$file" \
           | grep -E '^\+([^+]|$)' \
           | sed 's/^+//' || true)
  [[ -z "$added" ]] && continue

  for pattern in "${PATTERNS[@]}"; do
    matches=$(echo "$added" | grep -E "$pattern" || true)
    [[ -z "$matches" ]] && continue

    # Filter out lines marked with the inline allow comment.
    while IFS= read -r line; do
      if echo "$line" | grep -qF "$NOQA_MARKER"; then
        continue
      fi
      # Strip the leading '+' for display.
      VIOLATIONS+=("$file: $(echo "$line" | sed 's/^+//' | head -c 200)")
    done <<<"$matches"
  done
done

if [[ ${#VIOLATIONS[@]} -eq 0 ]]; then
  exit 0
fi

cat >&2 <<EOF

╔════════════════════════════════════════════════════════════════╗
║  Commit blocked: no-API-key policy violation                   ║
╚════════════════════════════════════════════════════════════════╝

The following lines reference the Anthropic API directly:

EOF

for v in "${VIOLATIONS[@]}"; do
  echo "  → $v" >&2
done

cat >&2 <<EOF

Why this is blocked:
  This repo uses the \`claude\` CLI (Claude Pro/Max subscription).
  Calling the API directly bills per token — bypassing the subscription
  and creating a surprise bill.

How to fix:
  - Replace API calls with \`claude -p\` invocations
  - Replace anthropic SDK imports with subprocess calls to the claude CLI
  - See docs/no-api-key-policy.md in claude-utils

Legitimate exceptions:
  - Documentation explaining the policy → file path contains "no-api-key-policy"
  - Example / template files → *.example, *.template
  - Single-line waiver → append: # noqa: anthropic-api
  - Last resort: git commit --no-verify (leaves a trail; avoid for code)

EOF

exit 1
