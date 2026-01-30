# Setup Guide for New Machines

This guide helps you set up Claude Code utilities on a new computer.

## Quick Setup (Automated)

Run this one-liner on your new machine:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/ERaith/claude-utils/master/setup.sh)
```

**Or manually:**

```bash
git clone https://github.com/ERaith/claude-utils.git ~/claude-utils
cd ~/claude-utils
./setup.sh
```

## What the Setup Script Does

1. **Clones the repository** to `~/claude-utils`
2. **Installs statusline script** to `~/scripts/claude-statusline.sh`
3. **Configures Claude settings** (creates or updates `~/.claude/settings.local.json`)
4. Sets up the status line to show:
   - Current Claude model
   - Working directory
   - Git branch (if in a repo)
   - Dirty indicator (*) for uncommitted changes

## Manual Setup

If you prefer to set up manually:

### 1. Clone the Repository
```bash
git clone https://github.com/ERaith/claude-utils.git ~/claude-utils
```

### 2. Install Statusline Script
```bash
mkdir -p ~/scripts
cp ~/claude-utils/scripts/claude-statusline.sh ~/scripts/
chmod +x ~/scripts/claude-statusline.sh
```

### 3. Configure Claude Settings

Edit or create `~/.claude/settings.local.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/home/YOUR_USERNAME/scripts/claude-statusline.sh"
  },
  "outputStyle": "Explanatory"
}
```

Replace `YOUR_USERNAME` with your actual username.

## Keeping Updated

Run the update script to pull latest changes and sync configurations:

```bash
~/claude-utils/update.sh
```

This will:
- Pull latest changes from GitHub
- Show what changed
- Update your statusline script
- Notify you of new scripts or agents

## What You Get

After setup, you'll have:

- **📊 Status line** showing model, directory, and git info
- **🩺 Saved agents** like torrent-doctor (see `agents/README.md`)
- **📚 Quick commands** reference (see `docs/quick-commands.md`)
- **🔄 Easy updates** with `update.sh`

## Troubleshooting

### Status line not showing?
1. Restart Claude Code
2. Check script exists: `ls -l ~/scripts/claude-statusline.sh`
3. Test manually: `~/scripts/claude-statusline.sh`

### Permission errors?
```bash
chmod +x ~/scripts/claude-statusline.sh
```

### Settings not applying?
Make sure the path in `~/.claude/settings.local.json` matches your username:
```bash
sed -i "s|/home/.*scripts|$HOME/scripts|g" ~/.claude/settings.local.json
```

---

**Need help?** Check the [main README](../README.md) or open an issue on GitHub.
