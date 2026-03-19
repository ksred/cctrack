package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coder/websocket"
	"github.com/ksred/cctrack/internal/calculator"
	"github.com/ksred/cctrack/internal/config"
	"github.com/ksred/cctrack/internal/hub"
	"github.com/ksred/cctrack/internal/store"
)

type API struct {
	store *store.Store
	hub   *hub.Hub
	cfg   *config.Config
}

func New(s *store.Store, h *hub.Hub, cfg *config.Config) *API {
	return &API{store: s, hub: h, cfg: cfg}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/summary", a.handleSummary)
	mux.HandleFunc("GET /api/v1/sessions", a.handleSessions)
	mux.HandleFunc("GET /api/v1/sessions/{id}", a.handleSession)
	mux.HandleFunc("GET /api/v1/recent", a.handleRecent)
	mux.HandleFunc("GET /api/v1/daily", a.handleDaily)
	mux.HandleFunc("GET /api/v1/settings", a.handleGetSettings)
	mux.HandleFunc("POST /api/v1/settings", a.handlePostSettings)
	mux.HandleFunc("GET /api/v1/projects", a.handleProjects)
	mux.HandleFunc("GET /api/v1/projects/monthly", a.handleProjectMonthly)
	mux.HandleFunc("GET /api/v1/rates", a.handleRates)
	mux.HandleFunc("GET /api/v1/models", a.handleModels)
	mux.HandleFunc("GET /api/v1/heatmap", a.handleHeatmap)
	mux.HandleFunc("GET /api/v1/sessions/{id}/requests", a.handleSessionRequests)
	mux.HandleFunc("GET /api/v1/ws", a.handleWS)
}

func (a *API) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := a.store.GetSummary()
	if err != nil {
		internalError(w, "summary", err)
		return
	}

	input, output, cacheRead, cacheWrite, err := a.store.GetTokenBreakdown()
	if err != nil {
		internalError(w, "token-breakdown", err)
		return
	}

	costBreakdown, err := a.store.GetCostBreakdown()
	if err != nil {
		internalError(w, "cost-breakdown", err)
		return
	}

	trends, err := a.store.GetTrends()
	if err != nil {
		internalError(w, "trends", err)
		return
	}

	resp := map[string]any{
		"today":     summary.Today,
		"week":      summary.Week,
		"month":     summary.Month,
		"projected": summary.Projected,
		"tokens": map[string]int64{
			"input":       input,
			"output":      output,
			"cache_read":  cacheRead,
			"cache_write": cacheWrite,
		},
		"cost_breakdown": costBreakdown,
		"trends":         trends,
		"budget":         a.cfg.MonthlyBudgetUSD,
	}
	writeJSON(w, resp)
}

func (a *API) handleSessions(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 25)
	offset := queryInt(r, "offset", 0)
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "cost"
	}
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "desc"
	}

	sessions, total, err := a.store.ListSessions(limit, offset, sort, dir)
	if err != nil {
		internalError(w, "sessions", err)
		return
	}

	writeJSON(w, map[string]any{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (a *API) handleSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := a.store.GetSession(id)
	if err != nil {
		http.Error(w, "session not found", 404)
		return
	}
	writeJSON(w, sess)
}

func (a *API) handleRecent(w http.ResponseWriter, r *http.Request) {
	n := queryInt(r, "n", 10)
	sessions, err := a.store.RecentSessions(n)
	if err != nil {
		internalError(w, "recent", err)
		return
	}
	writeJSON(w, sessions)
}

func (a *API) handleDaily(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 30)
	daily, err := a.store.GetDailySummary(days)
	if err != nil {
		internalError(w, "daily", err)
		return
	}
	writeJSON(w, daily)
}

func (a *API) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.cfg)
}

func (a *API) handlePostSettings(w http.ResponseWriter, r *http.Request) {
	var updates struct {
		MonthlyBudgetUSD   *float64 `json:"monthly_budget_usd"`
		OpenBrowserOnServe *bool    `json:"open_browser_on_serve"`
		LogDir             *string  `json:"log_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid JSON", 400)
		return
	}

	if updates.MonthlyBudgetUSD != nil {
		a.cfg.MonthlyBudgetUSD = *updates.MonthlyBudgetUSD
	}
	if updates.OpenBrowserOnServe != nil {
		a.cfg.OpenBrowserOnServe = *updates.OpenBrowserOnServe
	}
	if updates.LogDir != nil {
		if err := validateLogDir(*updates.LogDir); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		a.cfg.LogDir = *updates.LogDir
	}

	if err := a.cfg.Save(); err != nil {
		internalError(w, "save-settings", err)
		return
	}
	writeJSON(w, a.cfg)
}

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := a.store.GetProjects()
	if err != nil {
		internalError(w, "projects", err)
		return
	}
	writeJSON(w, projects)
}

func (a *API) handleProjectMonthly(w http.ResponseWriter, r *http.Request) {
	data, err := a.store.GetProjectMonthly()
	if err != nil {
		internalError(w, "project-monthly", err)
		return
	}
	writeJSON(w, data)
}

func (a *API) handleRates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, calculator.Rates)
}

func (a *API) handleModels(w http.ResponseWriter, r *http.Request) {
	models, err := a.store.GetModelBreakdown()
	if err != nil {
		internalError(w, "models", err)
		return
	}
	writeJSON(w, models)
}

func (a *API) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	cells, err := a.store.GetActivityHeatmap()
	if err != nil {
		internalError(w, "heatmap", err)
		return
	}
	writeJSON(w, cells)
}

func (a *API) handleSessionRequests(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	requests, err := a.store.GetSessionRequests(id)
	if err != nil {
		internalError(w, "session-requests", err)
		return
	}
	writeJSON(w, requests)
}

func (a *API) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			fmt.Sprintf("localhost:%d", a.cfg.Port),
			fmt.Sprintf("127.0.0.1:%d", a.cfg.Port),
		},
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	// Send initial summary snapshot
	summary, err := a.store.GetSummary()
	if err == nil {
		payload, _ := json.Marshal(summary)
		event := hub.Event{Type: "summary.updated", Payload: payload}
		data, _ := json.Marshal(event)
		conn.Write(r.Context(), websocket.MessageText, data)
	}

	a.hub.HandleConnection(conn)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// internalError logs the real error and returns a generic 500 to the client.
func internalError(w http.ResponseWriter, context string, err error) {
	log.Printf("API error [%s]: %v", context, err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// validateLogDir checks that the given path is a safe, existing directory
// within the user's home directory tree.
func validateLogDir(dir string) error {
	// Expand ~ prefix
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot resolve home directory")
		}
		dir = filepath.Join(home, dir[2:])
	}

	// Must be absolute after expansion
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("log_dir must be an absolute path")
	}

	// Clean the path and reject traversal
	cleaned := filepath.Clean(dir)
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot resolve home directory")
	}

	// Must be within home directory
	if !strings.HasPrefix(cleaned, home+string(filepath.Separator)) && cleaned != home {
		return fmt.Errorf("log_dir must be within your home directory")
	}

	// Check the directory exists
	info, err := os.Stat(cleaned)
	if err != nil {
		return fmt.Errorf("log_dir path does not exist or is not accessible")
	}
	if !info.IsDir() {
		return fmt.Errorf("log_dir path is not a directory")
	}

	return nil
}
