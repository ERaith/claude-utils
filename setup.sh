#!/bin/bash
# Claude Utils Setup Script
# Run this on a new machine to set up Claude Code utilities

set -e

echo "🚀 Claude Utils Setup"
echo "===================="
echo ""

# Check if already in claude-utils directory
if [[ "$(basename "$PWD")" == "claude-utils" ]]; then
    REPO_DIR="$PWD"
else
    # Clone the repository if not already cloned
    REPO_DIR="$HOME/claude-utils"
    if [ ! -d "$REPO_DIR" ]; then
        echo "📦 Cloning repository..."
        git clone https://github.com/ERaith/claude-utils.git "$REPO_DIR"
        cd "$REPO_DIR"
    else
        echo "✓ Repository already exists at $REPO_DIR"
        cd "$REPO_DIR"
        git pull
    fi
fi

echo ""
echo "📁 Repository location: $REPO_DIR"
echo ""

# Create scripts directory if it doesn't exist
mkdir -p ~/scripts

# Copy statusline script
echo "📊 Setting up statusline..."
cp "$REPO_DIR/scripts/claude-statusline.sh" ~/scripts/
chmod +x ~/scripts/claude-statusline.sh
echo "✓ Statusline script installed to ~/scripts/"

# Configure Claude settings
SETTINGS_FILE="$HOME/.claude/settings.local.json"
echo ""
echo "⚙️  Configuring Claude Code settings..."

if [ ! -f "$SETTINGS_FILE" ]; then
    # Create new settings file
    mkdir -p "$HOME/.claude"
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "statusLine": {
    "type": "command",
    "command": "/home/USER/scripts/claude-statusline.sh"
  },
  "permissions": {
    "allow": []
  },
  "outputStyle": "Explanatory"
}
EOF
    # Replace USER placeholder with actual username
    sed -i "s|/home/USER/|$HOME/|g" "$SETTINGS_FILE"
    echo "✓ Created new settings file"
else
    # Check if statusLine is already configured
    if grep -q '"statusLine"' "$SETTINGS_FILE"; then
        echo "⚠️  StatusLine already configured in $SETTINGS_FILE"
        echo "   Review manually if needed."
    else
        echo "⚠️  Settings file exists but no statusLine configured."
        echo "   Add this to your $SETTINGS_FILE:"
        echo ""
        echo '  "statusLine": {'
        echo '    "type": "command",'
        echo "    \"command\": \"$HOME/scripts/claude-statusline.sh\""
        echo '  },'
    fi
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "📚 Available resources:"
echo "   • Saved agents: $REPO_DIR/agents/README.md"
echo "   • Quick commands: $REPO_DIR/docs/quick-commands.md"
echo "   • Statusline config: $REPO_DIR/docs/statusline-config.md"
echo ""
echo "🔄 To update in the future, run: ~/claude-utils/update.sh"
echo ""
echo "🎉 Ready to use! Restart Claude Code to see the statusline."
