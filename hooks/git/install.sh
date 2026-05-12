#!/usr/bin/env bash
# install.sh — install claude-utils git hooks into the current repo.
#
# Idempotent: re-running updates symlinks, never destroys files.

set -euo pipefail

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "Error: not inside a git repository." >&2
  exit 1
fi

GIT_DIR=$(git rev-parse --git-dir)
HOOKS_DIR="$GIT_DIR/hooks"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$HOOKS_DIR"

install_hook() {
  local name="$1"
  local source_script="$2"
  local target="$HOOKS_DIR/$name"

  if [[ -e "$target" && ! -L "$target" ]]; then
    echo "  Skipping $name — existing non-symlink file at $target"
    echo "  Move it aside if you want claude-utils to manage it."
    return
  fi

  ln -sf "$source_script" "$target"
  chmod +x "$source_script"
  echo "  Installed $name → $source_script"
}

echo "Installing claude-utils git hooks into $HOOKS_DIR"
install_hook "pre-commit" "$SOURCE_DIR/pre-commit-no-api-key.sh"

echo ""
echo "Done. Test it with:"
echo "  git commit --allow-empty -m 'hook smoke test'"
