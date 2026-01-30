#!/bin/bash
# Claude Utils Update Script
# Pull latest changes and sync configurations

set -e

REPO_DIR="$HOME/claude-utils"

echo "🔄 Claude Utils Update"
echo "===================="
echo ""

# Check if repo exists
if [ ! -d "$REPO_DIR" ]; then
    echo "❌ Repository not found at $REPO_DIR"
    echo "   Run setup.sh first!"
    exit 1
fi

cd "$REPO_DIR"

# Store current commit
OLD_COMMIT=$(git rev-parse HEAD)

# Pull latest changes
echo "📥 Pulling latest changes..."
git pull

# Get new commit
NEW_COMMIT=$(git rev-parse HEAD)

if [ "$OLD_COMMIT" == "$NEW_COMMIT" ]; then
    echo "✓ Already up to date!"
else
    echo "✓ Updated to latest version"
    echo ""
    echo "📝 Recent changes:"
    git log --oneline "$OLD_COMMIT..$NEW_COMMIT" | head -5
fi

echo ""
echo "🔧 Syncing configurations..."

# Update statusline script
if [ -f "$REPO_DIR/scripts/claude-statusline.sh" ]; then
    mkdir -p ~/scripts
    cp "$REPO_DIR/scripts/claude-statusline.sh" ~/scripts/
    chmod +x ~/scripts/claude-statusline.sh
    echo "✓ Statusline script updated"
fi

# Check if any new scripts were added
NEW_SCRIPTS=$(find "$REPO_DIR/scripts" -name "*.sh" -type f ! -name "claude-statusline.sh" 2>/dev/null || true)
if [ -n "$NEW_SCRIPTS" ]; then
    echo ""
    echo "📜 New scripts available:"
    echo "$NEW_SCRIPTS" | while read -r script; do
        basename "$script"
    done
    echo ""
    echo "   Copy them to ~/scripts/ if needed"
fi

echo ""
echo "✅ Update complete!"
echo ""
echo "📚 Check docs for new features:"
echo "   • Agents: $REPO_DIR/agents/README.md"
echo "   • Commands: $REPO_DIR/docs/quick-commands.md"
echo ""
echo "💡 Tip: If settings changed, restart Claude Code to apply them."
