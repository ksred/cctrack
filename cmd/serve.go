package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ksred/cctrack/internal/api"
	"github.com/ksred/cctrack/internal/config"
	"github.com/ksred/cctrack/internal/credentials"
	"github.com/ksred/cctrack/internal/hub"
	"github.com/ksred/cctrack/internal/parser"
	"github.com/ksred/cctrack/internal/store"
	"github.com/ksred/cctrack/internal/usageprovider"
	"github.com/ksred/cctrack/internal/usagescheduler"
	"github.com/ksred/cctrack/internal/usagestate"
	"github.com/ksred/cctrack/internal/watcher"
	"github.com/spf13/cobra"
)

// WebFSFunc is set by main.go to provide the embedded web filesystem.
var WebFSFunc func() (fs.FS, error)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the dashboard server",
	Long:  "Parse logs, start the web dashboard, and watch for new activity.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Open store
		s, err := store.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer s.Close()

		// Initial parse
		p := parser.New(s)
		files, sessions, err := p.ParseAll(cfg.LogDir)
		if err != nil {
			log.Printf("Warning: initial parse failed: %v", err)
		} else {
			log.Printf("Parsed %d files, %d sessions", files, sessions)
		}

		// Start WebSocket hub
		h := hub.New()
		h.Start()
		defer h.Stop()

		// Auto-sync scheduler (F2 S2.2): the value is constructed up front
		// because both the watcher closure and the api handler need a
		// reference to the summaryProvider built on top of it. The scheduler
		// goroutine itself is spawned later, after ctx is ready.
		schedProvider := usageprovider.New()
		schedLogger := func(format string, args ...any) {
			log.Printf("usagescheduler: "+format, args...)
		}
		sched := usagescheduler.New(schedProvider, credentials.Load, s, schedLogger)

		// SummaryProvider is the SINGLE chokepoint for emitting augmented
		// summary payloads (per F2 S2.3 EM ruling chat msg 20621). All four
		// emission paths — REST /api/v1/summary, websocket-initial,
		// watcher broadcasts, scheduler broadcasts — call summaryProvider.Build
		// so the additive honest-state fields are populated consistently.
		summaryProvider := usagestate.NewSummaryProvider(s, sched)
		broadcastSummary := func() error {
			summary, err := summaryProvider.Build()
			if err != nil {
				return fmt.Errorf("get summary: %w", err)
			}
			payload, err := json.Marshal(summary)
			if err != nil {
				return fmt.Errorf("marshal summary: %w", err)
			}
			h.Broadcast("summary.updated", payload)
			return nil
		}

		// Coalesce broadcast-triggering events instead of rebuilding the
		// summary inline: with several live Claude Code sessions appending
		// JSONL, watcher flushes can arrive faster than a summary rebuild
		// completes, and unbounded inline rebuilds starve the API of CPU
		// (30s+ /api/v1/summary latency). A capacity-1 dirty flag plus a
		// paced drain loop bounds rebuilds to ~1/s while the trailing mark
		// guarantees the final state is always broadcast.
		summaryDirty := make(chan struct{}, 1)
		markSummaryDirty := func() {
			select {
			case summaryDirty <- struct{}{}:
			default:
			}
		}

		// Start watcher
		w, err := watcher.New(cfg.LogDir, 250*time.Millisecond, func(paths []string) {
			affected, err := p.ParseFiles(paths)
			if err != nil {
				log.Printf("Watcher parse error: %v", err)
				return
			}
			if len(affected) > 0 {
				// Broadcast updates
				for _, sid := range affected {
					sess, err := s.GetSession(sid)
					if err == nil {
						payload, _ := json.Marshal(sess)
						h.Broadcast("session.updated", payload)
					}
				}
				// Mark the summary dirty; the drain loop broadcasts via the
				// single augmentation chokepoint so honest-state fields are
				// present.
				markSummaryDirty()
			}
		})
		if err != nil {
			log.Printf("Warning: file watcher failed to start: %v", err)
		} else {
			w.Start()
			defer w.Stop()
		}

		// Setup HTTP server
		if WebFSFunc == nil {
			return fmt.Errorf("web filesystem not initialized")
		}
		webFS, err := WebFSFunc()
		if err != nil {
			return fmt.Errorf("loading embedded web assets: %w", err)
		}

		mux := http.NewServeMux()
		apiHandler := api.New(s, h, cfg, summaryProvider.Build, sched.SyncOnce)
		apiHandler.RegisterRoutes(mux)
		mux.Handle("/", api.SPAHandler(webFS))

		addr := fmt.Sprintf(":%d", cfg.Port)
		srv := &http.Server{Addr: addr, Handler: mux}

		// Open browser
		if cfg.OpenBrowserOnServe {
			go func() {
				time.Sleep(200 * time.Millisecond)
				openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
			}()
		}

		// Graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		// Drain loop for coalesced summary broadcasts. Paced at ~1/s: after
		// each rebuild it sleeps before checking the dirty flag again, so a
		// burst of watcher flushes costs one rebuild, not one per flush.
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-summaryDirty:
					if err := broadcastSummary(); err != nil {
						log.Printf("summary broadcast: %v", err)
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
				}
			}
		}()

		// Auto-sync scheduler runtime wiring (F2 S2.2 + S2.3):
		// the OnAnchorsUpdated callback broadcasts a fresh augmented
		// summary through the websocket hub when the scheduler writes
		// at least one anchor (per EM ruling chat msg 20591/20593).
		// Routes through the same coalesced chokepoint the watcher uses,
		// so honest-state fields are present.
		schedOnUpdate := func(_ context.Context) error {
			markSummaryDirty()
			return nil
		}
		sched.WithOnAnchorsUpdated(schedOnUpdate)
		schedDone := make(chan struct{})
		go func() {
			sched.Run(ctx)
			close(schedDone)
		}()
		// On shutdown the scheduler observes ctx.Done() inside Run and exits
		// cleanly; the deferred wait is bounded so a hung scheduler can't block
		// process exit indefinitely.
		defer func() {
			select {
			case <-schedDone:
			case <-time.After(5 * time.Second):
				log.Println("usagescheduler: did not exit within 5s of shutdown")
			}
		}()

		go func() {
			<-ctx.Done()
			log.Println("Shutting down...")
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutCtx)
		}()

		log.Printf("Dashboard: http://localhost:%d", cfg.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	},
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
