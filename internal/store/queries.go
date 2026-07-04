package store

import (
	"fmt"
	"time"

	"github.com/ksred/cctrack/internal/calculator"
)

type Summary struct {
	Window5h  WindowBucket `json:"window_5h"`
	Today     SpendBucket  `json:"today"`
	Window7d  WindowBucket `json:"window_7d"`
	Month     SpendBucket  `json:"month"`
	Projected float64      `json:"projected"`
}

// WindowBucket describes a rolling usage window (5h or 7d) computed by walking
// the request timestamp stream forward and opening a new window every time a
// request lands at-or-after the previous window's end. Cascading windows
// (each closes on its own clock, not on inactivity) — see GetWindowBucket.
type WindowBucket struct {
	Start        string   `json:"start"`         // ISO 8601 UTC
	End          string   `json:"end"`           // ISO 8601 UTC
	Cost         float64  `json:"cost"`          // total cost of requests in this window
	Tokens       int64    `json:"tokens"`        // total tokens
	RequestCount int      `json:"request_count"` // number of requests
	PrevCost     float64  `json:"prev_cost"`     // cost of the previous window
	PrevStart    string   `json:"prev_start"`    // ISO 8601 UTC; "" if no prev window
	// Cap is the user's effective spend cap for this window in USD, inferred
	// from the most recent sync against claude.ai (cost / pct). Nil when no
	// sync with a percentage exists yet — clients fall back to prev_cost.
	Cap *float64 `json:"cap,omitempty"`
	// Pct is the upstream-reported utilization percentage (0-100) for this
	// window from the latest anchor in active use. Authoritative for the
	// currently-authenticated account; clients should prefer this over the
	// cost/cap derivation when present. Nil when no fresh anchor is in use
	// (e.g., cascade fallback) or the anchor was synced without a percentage.
	Pct *float64 `json:"pct,omitempty"`
	// AnchorCost is observed_cost recorded on the currently-active anchor
	// at sync time. Together with AnchorCap, lets clients project bar fill
	// between syncs as Pct + (Cost - AnchorCost) / AnchorCap * 100, so the
	// bar grows with usage while staying calibrated against the same row
	// the upstream pct was reported for. Nil when no fresh anchor is in use.
	AnchorCost *float64 `json:"anchor_cost,omitempty"`
	// AnchorCap is inferred_cap on the currently-active anchor (cost/pct
	// from the same row, not GetLatestCap's walk-past-null fallback). Nil
	// when the anchor was written with anthropic_pct = 0 — in that case
	// clients can't extrapolate and should freeze the bar at Pct until the
	// next sync.
	AnchorCap *float64 `json:"anchor_cap,omitempty"`
	// LastSyncedAt is when the user last anchored this window from claude.ai.
	// Surfaced on the bar so a stale anchor is visible at a glance — sync
	// drift accumulates and re-syncs are how the user corrects it.
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
	// State is the per-window honest-state enum populated by
	// internal/usagestate.SummaryProvider.Build before any summary payload
	// is emitted (REST, WS-initial, watcher broadcast, scheduler broadcast).
	// One of: auto_fresh, auto_stale, token_expired, provider_unavailable,
	// manual_anchor, fallback_cascade, unknown. Nil when no augmentation
	// has been performed (e.g. tests reading raw GetSummary). UI matches
	// the string against its render branches; see F2 S2.3 evidence
	// requirements + LAD v0.5.
	State *string `json:"state,omitempty"`
}

type SpendBucket struct {
	Cost   float64 `json:"cost"`
	Tokens int64   `json:"tokens"`
}

type DailySpend struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

// GetSummary builds the dashboard summary. Both 5h and 7d windows are rolling
// (each window opens at the first event after the previous one expired). When
// the user has synced an anchor from claude.ai (a "I see N min left right
// now" entry), we use that to override the cascading detection — it closes
// the data-source-scope gap that the cascading inference can't bridge.
func (s *Store) GetSummary() (*Summary, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	todayUTC := todayStart.UTC().Format(time.RFC3339Nano)
	monthUTC := monthStart.UTC().Format(time.RFC3339Nano)

	summary := &Summary{}

	w5, err := s.windowFromAnchorOrCascade("5h", 5*time.Hour)
	if err != nil {
		return nil, err
	}
	summary.Window5h = w5

	w7, err := s.windowFromAnchorOrCascade("7d", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	summary.Window7d = w7

	// Aggregate from requests, not sessions: a session is a temporal *range*
	// with a single last_activity point; bucketing total_cost by that point
	// dumps the entire session's cost into one calendar day. Per-request
	// timestamps attribute cost to the day it was actually incurred.
	// "Today" / "this month" start at LOCAL midnight, converted to a UTC
	// bound in Go rather than via DATE(timestamp, 'localtime') in SQL: a
	// plain lexicographic comparison on the stored UTC strings lets SQLite
	// range-scan idx_requests_timestamp instead of converting every row.
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0),
		       COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_5m_tokens + cache_write_1h_tokens), 0)
		FROM requests WHERE timestamp >= ?`, todayUTC).Scan(&summary.Today.Cost, &summary.Today.Tokens)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0),
		       COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_5m_tokens + cache_write_1h_tokens), 0)
		FROM requests WHERE timestamp >= ?`, monthUTC).Scan(&summary.Month.Cost, &summary.Month.Tokens)
	if err != nil {
		return nil, err
	}

	// Projected: current month cost / days elapsed * days in month
	dayOfMonth := now.Day()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	if dayOfMonth > 0 && summary.Month.Cost > 0 {
		summary.Projected = summary.Month.Cost / float64(dayOfMonth) * float64(daysInMonth)
	}

	return summary, nil
}

// windowFromAnchorOrCascade prefers a fresh user-synced anchor for the given
// window type; falls back to cascading inference when no anchor exists or the
// anchored window has fully elapsed (in which case the user needs to re-sync).
//
// Either way, attaches the latest known cap (from the most recent anchor with
// a percentage, regardless of staleness) so the dashboard can show a
// cap-relative fill instead of the prev-window comparison.
func (s *Store) windowFromAnchorOrCascade(windowType string, duration time.Duration) (WindowBucket, error) {
	var bucket WindowBucket
	a, err := s.GetLatestAnchor(windowType)
	if err != nil {
		return WindowBucket{}, err
	}
	used := false
	if a != nil {
		syncedAt, perr := time.Parse(time.RFC3339Nano, a.SyncedAt)
		if perr == nil {
			anchoredEnd := syncedAt.Add(time.Duration(a.TimeLeftMinutes) * time.Minute)
			if time.Now().Before(anchoredEnd) {
				// Anchor still valid — use it.
				start := anchoredEnd.Add(-duration)
				b, e := s.computeAnchoredBucket(start, anchoredEnd, duration)
				if e != nil {
					return WindowBucket{}, e
				}
				bucket = b
				used = true
			}
		}
	}
	if !used {
		b, e := s.GetWindowBucket(duration)
		if e != nil {
			return WindowBucket{}, e
		}
		bucket = b
	}

	cap, err := s.GetLatestCap(windowType)
	if err == nil && cap != nil {
		bucket.Cap = cap
	}
	// Surface the upstream pct only when the anchor whose window we're
	// using is the same one that carries the pct. Anchors whose window has
	// already elapsed describe a different reference window and would
	// mislead the bar if surfaced here.
	if used && a.AnthropicPct != nil {
		v := *a.AnthropicPct
		bucket.Pct = &v
		// Pair the pct with the same-row cost and cap so clients can
		// extrapolate bar fill from the anchor moment without falling
		// back to cross-account-leaky GetLatestCap.
		ac := a.ObservedCost
		bucket.AnchorCost = &ac
		if a.InferredCap != nil {
			ic := *a.InferredCap
			bucket.AnchorCap = &ic
		}
	}
	if a != nil && a.SyncedAt != "" {
		ts := a.SyncedAt
		bucket.LastSyncedAt = &ts
	}
	return bucket, nil
}

// CostInRange sums request cost in [start, end). Half-open interval so
// adjacent windows don't double-count the boundary timestamp.
func (s *Store) CostInRange(start, end time.Time) (float64, error) {
	var cost float64
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0)
		FROM requests
		WHERE timestamp >= ? AND timestamp < ?`,
		start.UTC().Format(time.RFC3339Nano),
		end.UTC().Format(time.RFC3339Nano),
	).Scan(&cost)
	return cost, err
}

func (s *Store) computeAnchoredBucket(start, end time.Time, duration time.Duration) (WindowBucket, error) {
	startStr := start.UTC().Format(time.RFC3339Nano)
	endStr := end.UTC().Format(time.RFC3339Nano)
	prevStart := start.Add(-duration)
	prevStartStr := prevStart.UTC().Format(time.RFC3339Nano)

	bucket := WindowBucket{
		Start:     startStr,
		End:       endStr,
		PrevStart: prevStartStr,
	}
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0),
		       COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens
		                  + cache_write_5m_tokens + cache_write_1h_tokens), 0),
		       COUNT(*)
		FROM requests WHERE timestamp >= ? AND timestamp < ?`,
		startStr, endStr,
	).Scan(&bucket.Cost, &bucket.Tokens, &bucket.RequestCount); err != nil {
		return WindowBucket{}, err
	}
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0)
		FROM requests WHERE timestamp >= ? AND timestamp < ?`,
		prevStartStr, startStr,
	).Scan(&bucket.PrevCost); err != nil {
		return WindowBucket{}, err
	}
	return bucket, nil
}

// GetWindowBucket walks the request timestamps in order, opens a new rolling
// window of `duration` when a request arrives at-or-after the previous
// window's end, and returns the most-recent window's totals plus the prior
// window's cost (for trend comparisons).
func (s *Store) GetWindowBucket(duration time.Duration) (WindowBucket, error) {
	rows, err := s.db.Query(`
		SELECT timestamp, cost,
			input_tokens + output_tokens + cache_read_tokens
				+ cache_write_5m_tokens + cache_write_1h_tokens
		FROM requests ORDER BY timestamp ASC`)
	if err != nil {
		return WindowBucket{}, err
	}
	defer rows.Close()

	type win struct {
		start  time.Time
		cost   float64
		tokens int64
		count  int
	}
	var windows []win
	var currentEnd time.Time

	for rows.Next() {
		var tsStr string
		var cost float64
		var tokens int64
		if err := rows.Scan(&tsStr, &cost, &tokens); err != nil {
			return WindowBucket{}, err
		}
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			ts, err = time.Parse(time.RFC3339, tsStr)
			if err != nil {
				continue
			}
		}
		if currentEnd.IsZero() || !ts.Before(currentEnd) {
			windows = append(windows, win{start: ts})
			currentEnd = ts.Add(duration)
		}
		w := &windows[len(windows)-1]
		w.cost += cost
		w.tokens += tokens
		w.count++
	}

	if len(windows) == 0 {
		return WindowBucket{}, nil
	}
	cur := windows[len(windows)-1]
	out := WindowBucket{
		Start:        cur.start.UTC().Format(time.RFC3339Nano),
		End:          cur.start.Add(duration).UTC().Format(time.RFC3339Nano),
		Cost:         cur.cost,
		Tokens:       cur.tokens,
		RequestCount: cur.count,
	}
	if len(windows) >= 2 {
		prev := windows[len(windows)-2]
		out.PrevCost = prev.cost
		out.PrevStart = prev.start.UTC().Format(time.RFC3339Nano)
	}
	return out, nil
}

func (s *Store) GetDailySummary(days int) ([]DailySpend, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Aggregate from requests, not sessions: a session has a single
	// last_activity timestamp, so summing sessions.total_cost grouped by
	// last_activity dumps every day of a multi-day session into the day it
	// last saw a request. Per-request timestamps attribute cost to the day
	// it was actually incurred.
	//
	// Buckets are one CostInRange per local day (local midnight bounds
	// computed in Go, so DST-length days stay correct) rather than a single
	// GROUP BY DATE(timestamp, 'localtime'): each range SUM rides
	// idx_requests_timestamp, where the GROUP BY ran a per-row timezone
	// conversion over every request in the window. Zero-cost days come out
	// naturally as zero-sum ranges.
	var daily []DailySpend
	for i := days; i >= 0; i-- {
		dayStart := todayStart.AddDate(0, 0, -i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		cost, err := s.CostInRange(dayStart, dayEnd)
		if err != nil {
			return nil, err
		}
		daily = append(daily, DailySpend{Date: dayStart.Format("2006-01-02"), Cost: cost})
	}
	return daily, nil
}

func (s *Store) TopSessions(n int) ([]Session, error) {
	rows, err := s.db.Query(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read,
		total_cache_write_5m, total_cache_write_1h, total_cost
		FROM sessions ORDER BY total_cost DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
			&sess.StartedAt, &sess.LastActivity,
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead,
			&sess.TotalCacheWrite5m, &sess.TotalCacheWrite1h,
			&sess.TotalCost); err != nil {
			return nil, err
		}
		sess.TotalCacheWrite = sess.TotalCacheWrite5m + sess.TotalCacheWrite1h
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *Store) RecentSessions(n int) ([]Session, error) {
	rows, err := s.db.Query(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read,
		total_cache_write_5m, total_cache_write_1h, total_cost
		FROM sessions ORDER BY last_activity DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
			&sess.StartedAt, &sess.LastActivity,
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead,
			&sess.TotalCacheWrite5m, &sess.TotalCacheWrite1h,
			&sess.TotalCost); err != nil {
			return nil, err
		}
		sess.TotalCacheWrite = sess.TotalCacheWrite5m + sess.TotalCacheWrite1h
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

type ProjectSummary struct {
	Project        string  `json:"project"`
	SessionCount   int     `json:"session_count"`
	TotalCost      float64 `json:"total_cost"`
	TotalTokens    int64   `json:"total_tokens"`
	TotalInput     int64   `json:"total_input"`
	TotalOutput    int64   `json:"total_output"`
	TotalCacheRead int64   `json:"total_cache_read"`
	TotalCacheWrite int64  `json:"total_cache_write"`
	LastActivity   string  `json:"last_activity"`
}

type ProjectMonthly struct {
	Project string  `json:"project"`
	Month   string  `json:"month"`
	Cost    float64 `json:"cost"`
}

// ProjectGroup is one row in the grouped Sessions view: a project with its
// roll-up stats, optionally restricted to sessions active on a given local day.
type ProjectGroup struct {
	Project      string  `json:"project"`
	SessionCount int     `json:"session_count"`
	TotalCost    float64 `json:"total_cost"`
	TotalTokens  int64   `json:"total_tokens"`
	StartedAt    string  `json:"started_at"`    // MIN across child sessions
	LastActivity string  `json:"last_activity"` // MAX across child sessions
}

var projectGroupSortColumns = map[string]string{
	"cost":    "total_cost",
	"date":    "last_activity",
	"started": "started_at",
	"tokens":  "total_tokens",
	"project": "project",
}

// GetProjectGroups returns project-level rollups for the Sessions grouped view.
// When date is "" all projects are returned; otherwise only projects with at
// least one session whose local-day `last_activity` equals `date`. Roll-up
// totals are session-lifetime — NOT day-scoped. This is the canonical
// browse view (Invariant B) and intentionally differs from request-day
// totals; for day-scoped per-request spend (e.g. the daily-bar drilldown),
// use GetDayDrilldown / `/api/v1/day-drilldown`.
func (s *Store) GetProjectGroups(date, sortBy, sortDir string) ([]ProjectGroup, error) {
	col, ok := projectGroupSortColumns[sortBy]
	if !ok {
		col = "last_activity"
	}
	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}

	whereClause := ""
	args := []any{}
	if date != "" {
		whereClause = "WHERE DATE(last_activity, 'localtime') = ?"
		args = append(args, date)
	}

	query := fmt.Sprintf(`
		SELECT project,
			COUNT(*) as session_count,
			SUM(total_cost) as total_cost,
			SUM(total_input + total_output + total_cache_read + total_cache_write_5m + total_cache_write_1h) as total_tokens,
			MIN(started_at) as started_at,
			MAX(last_activity) as last_activity
		FROM sessions
		%s
		GROUP BY project
		ORDER BY %s %s`, whereClause, col, dir)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []ProjectGroup
	for rows.Next() {
		var g ProjectGroup
		if err := rows.Scan(&g.Project, &g.SessionCount, &g.TotalCost, &g.TotalTokens,
			&g.StartedAt, &g.LastActivity); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (s *Store) GetProjects() ([]ProjectSummary, error) {
	rows, err := s.db.Query(`
		SELECT project,
			COUNT(*) as session_count,
			SUM(total_cost) as total_cost,
			SUM(total_input + total_output + total_cache_read + total_cache_write_5m + total_cache_write_1h) as total_tokens,
			SUM(total_input) as total_input,
			SUM(total_output) as total_output,
			SUM(total_cache_read) as total_cache_read,
			SUM(total_cache_write_5m + total_cache_write_1h) as total_cache_write,
			MAX(last_activity) as last_activity
		FROM sessions
		GROUP BY project
		ORDER BY total_cost DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ProjectSummary
	for rows.Next() {
		var p ProjectSummary
		if err := rows.Scan(&p.Project, &p.SessionCount, &p.TotalCost, &p.TotalTokens,
			&p.TotalInput, &p.TotalOutput, &p.TotalCacheRead, &p.TotalCacheWrite,
			&p.LastActivity); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// GetProjectsSpendInRange returns spend per project over [bounds.Start,
// bounds.End), descending by cost. Drives the "Spend by Project" donut on
// the Overview with whichever range the user picks.
//
// Joins requests → sessions so we attribute each request's cost to the day
// it was actually incurred (per-request timestamp), while grouping by the
// project that owns the parent session. A zero Start means no lower bound
// (all-time).
func (s *Store) GetProjectsSpendInRange(bounds TimeBounds) ([]ProjectMonthly, error) {
	where := "1=1"
	args := []any{}
	if !bounds.Start.IsZero() {
		where += " AND r.timestamp >= ?"
		args = append(args, bounds.Start.UTC().Format(time.RFC3339Nano))
	}
	where += " AND r.timestamp < ?"
	args = append(args, bounds.End.UTC().Format(time.RFC3339Nano))

	q := fmt.Sprintf(`
		SELECT s.project, SUM(r.cost) as cost
		FROM requests r
		JOIN sessions s ON s.id = r.session_id
		WHERE %s
		GROUP BY s.project
		HAVING cost > 0
		ORDER BY cost DESC`, where)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []ProjectMonthly
	for rows.Next() {
		var pm ProjectMonthly
		if err := rows.Scan(&pm.Project, &pm.Cost); err != nil {
			return nil, err
		}
		data = append(data, pm)
	}
	return data, nil
}

func (s *Store) GetProjectMonthly() ([]ProjectMonthly, error) {
	// Join requests → sessions so cross-month sessions are split by request
	// timestamp into the months they actually spanned, not lumped into the
	// month of last_activity.
	rows, err := s.db.Query(`
		SELECT s.project,
			STRFTIME('%Y-%m', r.timestamp, 'localtime') as month,
			SUM(r.cost) as cost
		FROM requests r
		JOIN sessions s ON s.id = r.session_id
		WHERE DATE(r.timestamp, 'localtime') >= DATE('now', '-6 months', 'localtime')
		GROUP BY s.project, month
		ORDER BY month ASC, cost DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []ProjectMonthly
	for rows.Next() {
		var pm ProjectMonthly
		if err := rows.Scan(&pm.Project, &pm.Month, &pm.Cost); err != nil {
			return nil, err
		}
		data = append(data, pm)
	}
	return data, nil
}

func (s *Store) GetTokenBreakdown() (input, output, cacheRead, cacheWrite int64, err error) {
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(total_input), 0),
		       COALESCE(SUM(total_output), 0),
		       COALESCE(SUM(total_cache_read), 0),
		       COALESCE(SUM(total_cache_write_5m + total_cache_write_1h), 0)
		FROM sessions`).Scan(&input, &output, &cacheRead, &cacheWrite)
	return
}

type CostByType struct {
	InputCost      float64 `json:"input_cost"`
	OutputCost     float64 `json:"output_cost"`
	CacheReadCost  float64 `json:"cache_read_cost"`
	CacheWriteCost float64 `json:"cache_write_cost"`
}

// GetCostBreakdownInRange computes input/output/cache-read/cache-write
// dollar totals over [bounds.Start, bounds.End). Walks per-request rows so
// each model's own rate card is applied (cache-write costs differ between
// Opus 4.7 and Haiku, etc.) — summing pre-calculated total_cost wouldn't
// give us the breakdown by token type.
func (s *Store) GetCostBreakdownInRange(bounds TimeBounds) (*CostByType, error) {
	where := "model != ''"
	args := []any{}
	if !bounds.Start.IsZero() {
		where += " AND timestamp >= ?"
		args = append(args, bounds.Start.UTC().Format(time.RFC3339Nano))
	}
	where += " AND timestamp < ?"
	args = append(args, bounds.End.UTC().Format(time.RFC3339Nano))

	q := fmt.Sprintf(`
		SELECT model, input_tokens, output_tokens, cache_read_tokens,
			cache_write_5m_tokens, cache_write_1h_tokens
		FROM requests WHERE %s`, where)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &CostByType{}
	for rows.Next() {
		var model string
		var inp, out, cr, cw5m, cw1h int64
		if err := rows.Scan(&model, &inp, &out, &cr, &cw5m, &cw1h); err != nil {
			return nil, err
		}
		cb := calculator.Calculate(model, calculator.TokenUsage{
			InputTokens:        inp,
			OutputTokens:       out,
			CacheReadTokens:    cr,
			CacheWrite5mTokens: cw5m,
			CacheWrite1hTokens: cw1h,
		})
		result.InputCost += cb.InputCost
		result.OutputCost += cb.OutputCost
		result.CacheReadCost += cb.CacheReadCost
		result.CacheWriteCost += cb.CacheWriteCost
	}
	return result, nil
}

// --- Feature: Model Usage Breakdown ---

type ModelSummary struct {
	Model        string  `json:"model"`
	Family       string  `json:"family"`
	SessionCount int     `json:"session_count"`
	TotalCost    float64 `json:"total_cost"`
	TotalTokens  int64   `json:"total_tokens"`
}

// GetModelBreakdownInRange returns per-model rollups computed from the
// per-request stream filtered by [bounds.Start, bounds.End). session_count
// is the number of distinct sessions touching the model in that window.
func (s *Store) GetModelBreakdownInRange(bounds TimeBounds) ([]ModelSummary, error) {
	where := "model != ''"
	args := []any{}
	if !bounds.Start.IsZero() {
		where += " AND timestamp >= ?"
		args = append(args, bounds.Start.UTC().Format(time.RFC3339Nano))
	}
	where += " AND timestamp < ?"
	args = append(args, bounds.End.UTC().Format(time.RFC3339Nano))

	q := fmt.Sprintf(`
		SELECT model,
			COUNT(DISTINCT session_id) as session_count,
			SUM(cost) as total_cost,
			SUM(input_tokens + output_tokens + cache_read_tokens
				+ cache_write_5m_tokens + cache_write_1h_tokens) as total_tokens
		FROM requests
		WHERE %s
		GROUP BY model
		ORDER BY total_cost DESC`, where)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ModelSummary
	for rows.Next() {
		var m ModelSummary
		if err := rows.Scan(&m.Model, &m.SessionCount, &m.TotalCost, &m.TotalTokens); err != nil {
			return nil, err
		}
		rates := calculator.GetRates(m.Model)
		m.Family = rates.Family
		results = append(results, m)
	}
	return results, nil
}

// --- Feature: Activity Heatmap ---

type HeatmapCell struct {
	Day  int     `json:"day"`  // 0=Sunday .. 6=Saturday
	Hour int     `json:"hour"` // 0..23
	Cost float64 `json:"cost"`
}

func (s *Store) GetActivityHeatmap() ([]HeatmapCell, error) {
	// Bucket per request, not per session: the prior session-based query put
	// every request of a multi-hour session into a single (dow, hour) cell —
	// the hour of last_activity — making long sessions look like single-hour
	// spikes. Per-request timestamps spread cost across the cells where the
	// work actually happened.
	rows, err := s.db.Query(`
		SELECT CAST(STRFTIME('%w', timestamp, 'localtime') AS INTEGER) as dow,
			CAST(STRFTIME('%H', timestamp, 'localtime') AS INTEGER) as hour,
			SUM(cost) as cost
		FROM requests
		WHERE timestamp != ''
		GROUP BY dow, hour
		ORDER BY dow, hour`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cells []HeatmapCell
	for rows.Next() {
		var c HeatmapCell
		if err := rows.Scan(&c.Day, &c.Hour, &c.Cost); err != nil {
			return nil, err
		}
		cells = append(cells, c)
	}
	return cells, nil
}

// --- Feature: Cost Velocity / Trend Comparison ---

type Trends struct {
	PrevDayCost   float64 `json:"prev_day_cost"`
	PrevMonthCost float64 `json:"prev_month_cost"`
}

func (s *Store) GetTrends() (*Trends, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	prevMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	t := &Trends{}

	// Previous day cost (yesterday) — sums per-request cost so multi-day
	// sessions don't spill across day boundaries. Buckets are local-calendar
	// days expressed as UTC bounds in Go (not DATE(timestamp, 'localtime')
	// in SQL) so the comparison range-scans idx_requests_timestamp instead
	// of converting every row.
	s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0)
		FROM requests WHERE timestamp >= ? AND timestamp < ?`,
		yesterdayStart.UTC().Format(time.RFC3339Nano),
		todayStart.UTC().Format(time.RFC3339Nano)).Scan(&t.PrevDayCost)

	// Previous month cost — same per-request aggregation.
	s.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0)
		FROM requests WHERE timestamp >= ? AND timestamp < ?`,
		prevMonthStart.UTC().Format(time.RFC3339Nano),
		monthStart.UTC().Format(time.RFC3339Nano)).Scan(&t.PrevMonthCost)

	return t, nil
}
