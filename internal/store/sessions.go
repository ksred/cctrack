package store

import "fmt"

type Session struct {
	ID             string  `json:"id"`
	Project        string  `json:"project"`
	Slug           string  `json:"slug"`
	Model          string  `json:"model"`
	StartedAt      string  `json:"started_at"`
	LastActivity   string  `json:"last_activity"`
	TotalInput     int64   `json:"total_input"`
	TotalOutput    int64   `json:"total_output"`
	TotalCacheRead int64   `json:"total_cache_read"`
	TotalCacheWrite int64  `json:"total_cache_write"`
	TotalCost      float64 `json:"total_cost"`
}

func (s *Session) TotalTokens() int64 {
	return s.TotalInput + s.TotalOutput + s.TotalCacheRead + s.TotalCacheWrite
}

type SessionDelta struct {
	ID             string
	Project        string
	Slug           string
	Model          string
	Timestamp      string
	DeltaInput     int64
	DeltaOutput    int64
	DeltaCacheRead int64
	DeltaCacheWrite int64
	DeltaCost      float64
}

// UpsertSession adds token deltas to an existing session or creates a new one.
// Token counts are ADDITIVE — new values add to existing totals.
func (s *Store) UpsertSession(d SessionDelta) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, project, slug, model, started_at, last_activity,
			total_input, total_output, total_cache_read, total_cache_write, total_cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			slug          = CASE WHEN excluded.slug != '' THEN excluded.slug ELSE sessions.slug END,
			model         = CASE WHEN excluded.model != '' THEN excluded.model ELSE sessions.model END,
			last_activity = CASE WHEN excluded.last_activity > sessions.last_activity THEN excluded.last_activity ELSE sessions.last_activity END,
			total_input   = sessions.total_input   + excluded.total_input,
			total_output  = sessions.total_output  + excluded.total_output,
			total_cache_read  = sessions.total_cache_read  + excluded.total_cache_read,
			total_cache_write = sessions.total_cache_write + excluded.total_cache_write,
			total_cost    = sessions.total_cost    + excluded.total_cost
	`, d.ID, d.Project, d.Slug, d.Model, d.Timestamp, d.Timestamp,
		d.DeltaInput, d.DeltaOutput, d.DeltaCacheRead, d.DeltaCacheWrite, d.DeltaCost)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
		FROM sessions WHERE id = ?`, id)
	sess := &Session{}
	err := row.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
		&sess.StartedAt, &sess.LastActivity,
		&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
		&sess.TotalCost)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// --- Request-level tracking ---

type RequestRecord struct {
	RequestID       string  `json:"request_id"`
	SessionID       string  `json:"session_id"`
	Timestamp       string  `json:"timestamp"`
	Model           string  `json:"model"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	CacheReadTokens int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64  `json:"cache_write_tokens"`
	Cost            float64 `json:"cost"`
}

func (s *Store) UpsertRequest(r RequestRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO requests (request_id, session_id, timestamp, model,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(request_id) DO UPDATE SET
			timestamp = excluded.timestamp,
			model = excluded.model,
			input_tokens = excluded.input_tokens,
			output_tokens = excluded.output_tokens,
			cache_read_tokens = excluded.cache_read_tokens,
			cache_write_tokens = excluded.cache_write_tokens,
			cost = excluded.cost
	`, r.RequestID, r.SessionID, r.Timestamp, r.Model,
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.Cost)
	return err
}

func (s *Store) GetSessionRequests(sessionID string) ([]RequestRecord, error) {
	rows, err := s.db.Query(`
		SELECT request_id, session_id, timestamp, model,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost
		FROM requests WHERE session_id = ?
		ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []RequestRecord
	for rows.Next() {
		var r RequestRecord
		if err := rows.Scan(&r.RequestID, &r.SessionID, &r.Timestamp, &r.Model,
			&r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheWriteTokens,
			&r.Cost); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return recs, nil
}

var allowedSortColumns = map[string]string{
	"cost":       "total_cost",
	"date":       "last_activity",
	"started":    "started_at",
	"tokens":     "(total_input + total_output + total_cache_read + total_cache_write)",
	"model":      "model",
	"project":    "project",
}

func (s *Store) ListSessions(limit, offset int, sortBy, sortDir string) ([]Session, int, error) {
	col, ok := allowedSortColumns[sortBy]
	if !ok {
		col = "total_cost"
	}
	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}

	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
		FROM sessions ORDER BY %s %s LIMIT ? OFFSET ?`, col, dir)

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
			&sess.StartedAt, &sess.LastActivity,
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
			&sess.TotalCost); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}
