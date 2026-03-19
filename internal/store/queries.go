package store

import (
	"sort"
	"time"

	"github.com/ksred/cctrack/internal/calculator"
)

func sortProjectMonthly(data []ProjectMonthly) {
	sort.Slice(data, func(i, j int) bool {
		if data[i].Month != data[j].Month {
			return data[i].Month < data[j].Month
		}
		return data[i].Cost > data[j].Cost
	})
}

type Summary struct {
	Today     SpendBucket `json:"today"`
	Week      SpendBucket `json:"week"`
	Month     SpendBucket `json:"month"`
	Projected float64     `json:"projected"`
}

type SpendBucket struct {
	Cost   float64 `json:"cost"`
	Tokens int64   `json:"tokens"`
}

type DailySpend struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

func (s *Store) GetSummary() (*Summary, error) {
	now := time.Now()
	todayStart := startOfDay(now).Format(time.RFC3339)
	tomorrowStart := startOfDay(now).AddDate(0, 0, 1).Format(time.RFC3339)
	weekAgo := startOfDay(now).AddDate(0, 0, -7).Format(time.RFC3339)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.RFC3339)

	summary := &Summary{}

	// Today: sessions active between local midnight and local tomorrow midnight
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0),
		       COALESCE(SUM(total_input + total_output + total_cache_read + total_cache_write), 0)
		FROM sessions WHERE last_activity >= ? AND last_activity < ?`,
		todayStart, tomorrowStart).Scan(&summary.Today.Cost, &summary.Today.Tokens)
	if err != nil {
		return nil, err
	}

	// This week
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0),
		       COALESCE(SUM(total_input + total_output + total_cache_read + total_cache_write), 0)
		FROM sessions WHERE last_activity >= ?`, weekAgo).Scan(&summary.Week.Cost, &summary.Week.Tokens)
	if err != nil {
		return nil, err
	}

	// This month
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0),
		       COALESCE(SUM(total_input + total_output + total_cache_read + total_cache_write), 0)
		FROM sessions WHERE last_activity >= ?`, monthStart).Scan(&summary.Month.Cost, &summary.Month.Tokens)
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

func (s *Store) GetDailySummary(days int) ([]DailySpend, error) {
	now := time.Now()
	since := startOfDay(now).AddDate(0, 0, -days).Format(time.RFC3339)

	// Query raw timestamps and costs — no SQLite date functions
	rows, err := s.db.Query(`
		SELECT last_activity, total_cost
		FROM sessions
		WHERE last_activity >= ?`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Bucket costs by local date in Go
	result := make(map[string]float64)
	for rows.Next() {
		var ts string
		var cost float64
		if err := rows.Scan(&ts, &cost); err != nil {
			return nil, err
		}
		localDate := parseLocalTime(ts).Format("2006-01-02")
		result[localDate] += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build a complete date range with zero-filled gaps
	var daily []DailySpend
	for i := days; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		daily = append(daily, DailySpend{Date: d, Cost: result[d]})
	}
	return daily, nil
}

func (s *Store) TopSessions(n int) ([]Session, error) {
	rows, err := s.db.Query(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
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
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
			&sess.TotalCost); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *Store) RecentSessions(n int) ([]Session, error) {
	rows, err := s.db.Query(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
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
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
			&sess.TotalCost); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (s *Store) GetProjects() ([]ProjectSummary, error) {
	rows, err := s.db.Query(`
		SELECT project,
			COUNT(*) as session_count,
			SUM(total_cost) as total_cost,
			SUM(total_input + total_output + total_cache_read + total_cache_write) as total_tokens,
			SUM(total_input) as total_input,
			SUM(total_output) as total_output,
			SUM(total_cache_read) as total_cache_read,
			SUM(total_cache_write) as total_cache_write,
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *Store) GetProjectMonthly() ([]ProjectMonthly, error) {
	sixMonthsAgo := time.Now().AddDate(0, -6, 0).Format(time.RFC3339)

	// Query raw timestamps — bucket by month in Go
	rows, err := s.db.Query(`
		SELECT project, last_activity, total_cost
		FROM sessions
		WHERE last_activity >= ?`, sixMonthsAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate by project + local month
	type key struct{ project, month string }
	agg := make(map[key]float64)
	for rows.Next() {
		var project, ts string
		var cost float64
		if err := rows.Scan(&project, &ts, &cost); err != nil {
			return nil, err
		}
		month := parseLocalTime(ts).Format("2006-01")
		agg[key{project, month}] += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Flatten and sort
	var data []ProjectMonthly
	for k, cost := range agg {
		data = append(data, ProjectMonthly{Project: k.project, Month: k.month, Cost: cost})
	}
	// Sort by month ASC, then cost DESC
	sortProjectMonthly(data)
	return data, nil
}

func (s *Store) GetTokenBreakdown() (input, output, cacheRead, cacheWrite int64, err error) {
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(total_input), 0),
		       COALESCE(SUM(total_output), 0),
		       COALESCE(SUM(total_cache_read), 0),
		       COALESCE(SUM(total_cache_write), 0)
		FROM sessions`).Scan(&input, &output, &cacheRead, &cacheWrite)
	return
}

type CostByType struct {
	InputCost      float64 `json:"input_cost"`
	OutputCost     float64 `json:"output_cost"`
	CacheReadCost  float64 `json:"cache_read_cost"`
	CacheWriteCost float64 `json:"cache_write_cost"`
}

func (s *Store) GetCostBreakdown() (*CostByType, error) {
	rows, err := s.db.Query(`
		SELECT model, total_input, total_output, total_cache_read, total_cache_write
		FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &CostByType{}
	for rows.Next() {
		var model string
		var inp, out, cr, cw int64
		if err := rows.Scan(&model, &inp, &out, &cr, &cw); err != nil {
			return nil, err
		}
		cb := calculator.Calculate(model, calculator.TokenUsage{
			InputTokens:      inp,
			OutputTokens:     out,
			CacheReadTokens:  cr,
			CacheWriteTokens: cw,
		})
		result.InputCost += cb.InputCost
		result.OutputCost += cb.OutputCost
		result.CacheReadCost += cb.CacheReadCost
		result.CacheWriteCost += cb.CacheWriteCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (s *Store) GetModelBreakdown() ([]ModelSummary, error) {
	rows, err := s.db.Query(`
		SELECT model,
			COUNT(*) as session_count,
			SUM(total_cost) as total_cost,
			SUM(total_input + total_output + total_cache_read + total_cache_write) as total_tokens
		FROM sessions
		WHERE model != ''
		GROUP BY model
		ORDER BY total_cost DESC`)
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
	if err := rows.Err(); err != nil {
		return nil, err
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
	// Query raw timestamps — bucket by day-of-week and hour in Go using local time
	rows, err := s.db.Query(`
		SELECT last_activity, total_cost
		FROM sessions
		WHERE last_activity != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type cellKey struct{ day, hour int }
	agg := make(map[cellKey]float64)
	for rows.Next() {
		var ts string
		var cost float64
		if err := rows.Scan(&ts, &cost); err != nil {
			return nil, err
		}
		t := parseLocalTime(ts)
		if t.IsZero() {
			continue
		}
		k := cellKey{day: int(t.Weekday()), hour: t.Hour()}
		agg[k] += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var cells []HeatmapCell
	for k, cost := range agg {
		cells = append(cells, HeatmapCell{Day: k.day, Hour: k.hour, Cost: cost})
	}
	// Sort by day, then hour for consistent output
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Day != cells[j].Day {
			return cells[i].Day < cells[j].Day
		}
		return cells[i].Hour < cells[j].Hour
	})
	return cells, nil
}

// --- Feature: Cost Velocity / Trend Comparison ---

type Trends struct {
	PrevDayCost  float64 `json:"prev_day_cost"`
	PrevWeekCost float64 `json:"prev_week_cost"`
	PrevMonthCost float64 `json:"prev_month_cost"`
}

func (s *Store) GetTrends() (*Trends, error) {
	now := time.Now()
	todayStart := startOfDay(now).Format(time.RFC3339)
	yesterdayStart := startOfDay(now).AddDate(0, 0, -1).Format(time.RFC3339)

	twoWeeksAgo := startOfDay(now).AddDate(0, 0, -14).Format(time.RFC3339)
	oneWeekAgo := startOfDay(now).AddDate(0, 0, -7).Format(time.RFC3339)

	prevMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()).Format(time.RFC3339)
	prevMonthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.RFC3339)

	t := &Trends{}

	// Previous day cost (yesterday midnight to today midnight, local time)
	s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0)
		FROM sessions WHERE last_activity >= ? AND last_activity < ?`,
		yesterdayStart, todayStart).Scan(&t.PrevDayCost)

	// Previous week cost (7-14 days ago)
	s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0)
		FROM sessions WHERE last_activity >= ? AND last_activity < ?`,
		twoWeeksAgo, oneWeekAgo).Scan(&t.PrevWeekCost)

	// Previous month cost
	s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0)
		FROM sessions WHERE last_activity >= ? AND last_activity < ?`,
		prevMonthStart, prevMonthEnd).Scan(&t.PrevMonthCost)

	return t, nil
}
