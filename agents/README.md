# Saved Claude Code Agents

This file tracks useful agents that can be resumed in the future.

## How to Resume an Agent

Use the Task tool with the `resume` parameter:
```
Task tool with subagent_type="general-purpose", resume="AGENT_ID"
```

Or ask Claude: "Resume agent AGENT_ID" or "Resume the transmission-optimizer agent"

---

## Agent Registry

### Transmission Optimizer
- **Agent ID**: `a6e5bf0`
- **Created**: 2026-01-30
- **Purpose**: Optimizes Transmission torrent client for NordVPN with closed port
- **Tasks**:
  1. Removes low-availability torrents
  2. Optimizes Transmission settings for closed-port operation
  3. Analyzes and prioritizes remaining torrents
- **Status**: Paused (waiting for user input on partial data handling)
- **Resume command**: Resume agent a6e5bf0

---

## Template for New Agents

```markdown
### [Agent Name]
- **Agent ID**: `xxxxxxx`
- **Created**: YYYY-MM-DD
- **Purpose**: [Brief description]
- **Tasks**: [What it does]
- **Status**: [Active/Paused/Completed]
- **Notes**: [Any important context]
```
