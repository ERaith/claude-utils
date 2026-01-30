# Claude Code Utilities

A collection of useful Claude Code agents, scripts, and configurations to supercharge your Claude CLI experience.

## 🚀 Quick Start (New Machine)

Run this on any new computer to set up everything:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/ERaith/claude-utils/master/setup.sh)
```

This installs:
- 📊 Status line (shows model, directory, git branch)
- 🩺 Saved agents (like torrent-doctor)
- 📚 Command references and documentation

## 🔄 Keeping Updated

Pull latest changes and sync configurations:

```bash
~/claude-utils/update.sh
```

## 📁 What's Inside

- **`agents/`** - Saved agent configurations and IDs for resuming useful agents
- **`scripts/`** - Useful bash scripts and automation tools
  - `claude-statusline.sh` - Shows model, directory, and git info
- **`docs/`** - Documentation and command references
  - Quick commands, statusline config, setup guide
- **`setup.sh`** - One-time setup for new machines
- **`update.sh`** - Pull updates and sync configurations

## 📚 Documentation

- **[Setup Guide](docs/setup-guide.md)** - Setting up on new machines
- **[Saved Agents](agents/README.md)** - Resume useful agents like torrent-doctor
- **[Quick Commands](docs/quick-commands.md)** - Frequently used commands
- **[Statusline Config](docs/statusline-config.md)** - Customize your status line

## 🩺 Featured Agents

### torrent-doctor (ID: `a6e5bf0`)
Diagnoses and fixes Transmission torrent download issues:
- Removes low-availability torrents
- Optimizes settings for closed-port operation
- Analyzes and prioritizes remaining torrents

**Resume with:** `"Resume agent a6e5bf0"` or `"Resume torrent-doctor"`

## 🤝 Contributing

This is a personal utilities repo, but feel free to fork and adapt for your own use!

---

**Created:** 2026-01-30
**Author:** ERaith
