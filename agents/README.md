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

### 🩺 torrent-doctor
- **Agent ID**: `a6e5bf0`
- **Alias**: `torrent-doctor`
- **Created**: 2026-01-30
- **Purpose**: Diagnoses and fixes Transmission torrent download issues for NordVPN with closed port
- **Tasks**:
  1. Removes low-availability torrents (cleans up the patient list)
  2. Optimizes Transmission settings for closed-port operation (prescribes treatment)
  3. Analyzes and prioritizes remaining torrents (triage system)
- **Status**: Paused (waiting for user input on partial data handling)
- **Resume command**: "Resume agent a6e5bf0" or "Resume torrent-doctor"

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
