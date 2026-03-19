package store

import (
	"time"

	"github.com/ksred/cctrack/internal/calculator"
)

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
	todayStr := now.Format("2006-01-02")
	weekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	summary := &Summary{}

	// Today
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0),
		       COALESCE(SUM(total_input + total_output + total_cache_read + total_cache_write), 0)
		FROM sessions WHERE last_activity >= ?`, todayStr).Scan(&summary.Today.Cost, &summary.Today.Tokens)
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
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT DATE(last_activity) as day, SUM(total_cost)
		FROM sessions
		WHERE last_activity >= ?
		GROUP BY day
		ORDER BY day ASC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build a complete date range with zero-filled gaps
	result := make(map[string]float64)
	for rows.Next() {
		var day string
		var cost float64
		if err := rows.Scan(&day, &cost); err != nil {
			return nil, err
		}
		result[day] = cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var daily []DailySpend
	for i := days; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		cost := result[d]
		daily = append(daily, DailySpend{Date: d, Cost: cost})
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
	rows, err := s.db.Query(`
		SELECT project,
			STRFTIME('%Y-%m', last_activity) as month,
			SUM(total_cost) as cost
		FROM sessions
		WHERE last_activity >= DATE('now', '-6 months')
		GROUP BY project, month
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
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
	rows, err := s.db.Query(`
		SELECT CAST(STRFTIME('%w', last_activity) AS INTEGER) as dow,
			CAST(STRFTIME('%H', last_activity, 'localtime') AS INTEGER) as hour,
			SUM(total_cost) as cost
		FROM sessions
		WHERE last_activity != ''
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
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
	todayStr := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	twoWeeksAgo := now.AddDate(0, 0, -14).Format("2006-01-02")
	oneWeekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")

	prevMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	prevMonthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	t := &Trends{}

	// Previous day cost (yesterday)
	s.db.QueryRow(`
		SELECT COALESCE(SUM(total_cost), 0)
		FROM sessions WHERE DATE(last_activity) >= ? AND DATE(last_activity) < ?`,
		yesterday, todayStr).Scan(&t.PrevDayCost)

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
