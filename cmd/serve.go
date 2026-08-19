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
	"github.com/ksred/cctrack/internal/hub"
	"github.com/ksred/cctrack/internal/parser"
	"github.com/ksred/cctrack/internal/store"
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
				// Broadcast summary update
				summary, err := s.GetSummary()
				if err == nil {
					payload, _ := json.Marshal(summary)
					h.Broadcast("summary.updated", payload)
				}
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
		apiHandler := api.New(s, h, cfg)
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
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
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
