```
cd web && bun run build && cd .. && go build -o cctrack . && ./cctrack serve
```

# cctrack

A cost tracker for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Parses your local JSONL logs, calculates spend per session/project/model, and serves a real-time dashboard — all from a single binary.

## Features

- **Cost tracking** — today, this week, this month, and projected monthly spend
- **Session explorer** — browse every Claude Code session with token and cost breakdowns
- **Project breakdown** — see spend grouped by project, with monthly trends
- **Model breakdown** — usage and cost per model (Opus, Sonnet, Haiku)
- **Activity heatmap** — visualize when you're using Claude Code most
- **Request timeline** — per-request token usage within each session
- **Real-time updates** — file watcher + WebSocket push when new activity is detected
- **Budget tracking** — set a monthly budget and see progress against it
- **Single binary** — Go CLI with an embedded Vue 3 SPA, no separate frontend server needed

## Installation

### From source

```bash
git clone https://github.com/ksred/cctrack.git
cd cctrack
cd web && npm install && npm run build && cd ..
go build -o cctrack .
```

### Go install

```bash
# Requires the web/dist directory to be pre-built
go install github.com/ksred/cctrack@latest
```

## Usage

### Start the dashboard

```bash
cctrack serve
```

Opens a web dashboard on `http://localhost:8877` with real-time cost tracking. Parses logs on startup and watches for new activity.

### Parse logs manually

```bash
cctrack parse
```

Scans `~/.claude/projects/` for JSONL log files and updates the SQLite database.

### Quick status

```bash
cctrack status
```

Prints today/week/month spend and your most expensive session to stdout.

### View configuration

```bash
cctrack config
```

## How it works

1. Claude Code writes JSONL logs to `~/.claude/projects/<project>/<session>.jsonl`
2. cctrack scans these files, extracts token usage from `assistant` messages, and deduplicates by `requestId`
3. Costs are calculated using Anthropic's published per-token rates for each model
4. Data is stored in a local SQLite database (`~/.config/cctrack/cctrack.db`)
5. The `serve` command starts an HTTP server with a REST API and embedded Vue SPA
6. A file watcher detects new log activity and pushes updates via WebSocket

## Configuration

Config is stored at `~/.config/cctrack/config.json`:

```json
{
  "log_dir": "~/.claude/projects",
  "db_path": "~/.config/cctrack/cctrack.db",
  "port": 8877,
  "monthly_budget_usd": 200,
  "open_browser_on_serve": true
}
```

All settings can also be changed from the dashboard's settings page.

## License

[MIT](LICENSE)
