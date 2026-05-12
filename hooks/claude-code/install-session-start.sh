#!/usr/bin/env bash
# install-session-start.sh
#
# Copies the session-start template into a user-customizable location and
# registers it in ~/.claude/settings.json.
#
# Idempotent. Re-running prints the current install path without overwriting
# a customized hook script.

set -euo pipefail

SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="$SOURCE_DIR/session-start.template.sh"

if [[ ! -f "$TEMPLATE" ]]; then
  echo "Error: template not found at $TEMPLATE" >&2
  exit 1
fi

CLAUDE_DIR="$HOME/.claude"
HOOKS_DIR="$CLAUDE_DIR/hooks"
TARGET="$HOOKS_DIR/session-start.sh"
SETTINGS="$CLAUDE_DIR/settings.json"

mkdir -p "$HOOKS_DIR"

if [[ -e "$TARGET" ]]; then
  echo "Hook already exists: $TARGET"
  echo "Not overwriting (you've likely customized it)."
else
  cp "$TEMPLATE" "$TARGET"
  chmod +x "$TARGET"
  echo "Installed: $TARGET"
  echo "Edit it to add project-specific context (look for CUSTOMIZE blocks)."
fi

# Register in settings.json (only if not already registered).
if [[ -f "$SETTINGS" ]] && grep -q '"SessionStart"' "$SETTINGS"; then
  echo "SessionStart already registered in $SETTINGS — not modifying."
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo ""
  echo "Couldn't auto-update $SETTINGS (python3 not found)."
  echo "Add this manually:"
  cat <<EOF
{
  "hooks": {
    "SessionStart": [{
      "matcher": "startup",
      "hooks": [{"type": "command", "command": "$TARGET"}]
    }]
  }
}
EOF
  exit 0
fi

python3 - "$SETTINGS" "$TARGET" <<'PYEOF'
import json, sys
from pathlib import Path

settings_path = Path(sys.argv[1])
hook_cmd = sys.argv[2]

if settings_path.exists():
    data = json.loads(settings_path.read_text() or "{}")
else:
    data = {}

hooks = data.setdefault("hooks", {})
session_start = hooks.setdefault("SessionStart", [])

# Don't duplicate if already present.
for entry in session_start:
    for h in entry.get("hooks", []):
        if h.get("command") == hook_cmd:
            print(f"Already registered in {settings_path}")
            sys.exit(0)

session_start.append({
    "matcher": "startup",
    "hooks": [{"type": "command", "command": hook_cmd}],
})

settings_path.write_text(json.dumps(data, indent=2))
print(f"Registered SessionStart hook in {settings_path}")
PYEOF
