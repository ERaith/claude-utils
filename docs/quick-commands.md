# Quick Commands Reference

## Docker Management

### Check Service Status
```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

### View Logs
```bash
docker logs -f --tail=100 <container-name>
```

### Restart Service
```bash
docker restart <container-name>
```

## Transmission Commands

### View All Torrents
```bash
transmission-remote -l
```

### Detailed Torrent Info
```bash
transmission-remote -t <torrent-id> -i
```

### Remove Torrent (keep data)
```bash
transmission-remote -t <torrent-id> -r
```

### Remove Torrent (delete data)
```bash
transmission-remote -t <torrent-id> -rad
```

## System Monitoring

### Bandwidth Status
```bash
/home/eraith/scripts/bandwidth-status.sh
```

### Network Stats
```bash
vnstat -l
```

### Storage Usage
```bash
df -h | grep -E '(Filesystem|/media|/home)'
```

## Claude Code

### Resume Last Session
```bash
claude --continue
```

### Resume Specific Session
```bash
claude --resume <session-id>
```

---

Last updated: 2026-01-30
