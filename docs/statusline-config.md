# Claude Code Status Line Configuration

## Current Configuration

The status line is configured in `~/.claude/settings.local.json` and runs `/home/eraith/scripts/claude-statusline.sh`

### What It Shows

The status line displays at the bottom of your terminal and includes:

1. **Model** - Current Claude model (sonnet/opus/haiku)
2. **Current Directory** - Your working directory (~ for home)
3. **Git Branch** - Current git branch (when in a repo)
4. **Dirty Indicator** - `*` appears if you have uncommitted changes

**Example Output:**
```
sonnet | ~/claude-utils | master*
```

## Configuration Details

### Location
- **Settings file**: `~/.claude/settings.local.json`
- **Script**: `/home/eraith/scripts/claude-statusline.sh`
- **Backup script**: `~/claude-utils/scripts/claude-statusline.sh`

### Settings JSON
```json
{
  "statusLine": {
    "type": "command",
    "command": "/home/eraith/scripts/claude-statusline.sh"
  }
}
```

## How It Works

The status line script runs every time your prompt is displayed, providing real-time context:

- **Git Awareness**: Automatically detects when you're in a git repository
- **Model Awareness**: Shows which Claude model is currently active
- **Directory Awareness**: Always shows your current working directory

## Customization

To modify what's displayed, edit `/home/eraith/scripts/claude-statusline.sh`. The script uses bash and standard git commands.

### Available Environment Variables
- `$CLAUDE_MODEL` - Current model being used
- Standard bash variables like `$PWD`, `$HOME`, etc.

## Troubleshooting

If the status line doesn't appear:
1. Restart Claude Code to reload settings
2. Check that the script is executable: `chmod +x /home/eraith/scripts/claude-statusline.sh`
3. Test the script manually: `/home/eraith/scripts/claude-statusline.sh`

---

**Configured**: 2026-01-30
**Last Updated**: 2026-01-30
