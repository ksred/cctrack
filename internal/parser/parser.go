package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ksred/cctrack/internal/calculator"
	"github.com/ksred/cctrack/internal/store"
)

type Parser struct {
	store *store.Store
}

func New(s *store.Store) *Parser {
	return &Parser{store: s}
}

// ParseAll discovers and parses all JSONL files in the log directory.
// Returns the number of files parsed and sessions affected.
func (p *Parser) ParseAll(logDir string) (int, int, error) {
	files, err := DiscoverFiles(logDir)
	if err != nil {
		return 0, 0, fmt.Errorf("discovering files: %w", err)
	}

	filesParsed := 0
	sessionsAffected := make(map[string]bool)

	for _, path := range files {
		sessions, err := p.ParseFile(path)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			continue
		}
		if len(sessions) > 0 {
			filesParsed++
			for _, s := range sessions {
				sessionsAffected[s] = true
			}
		}
	}

	return filesParsed, len(sessionsAffected), nil
}

// ParseFile reads new data from a single JSONL file (from last known offset).
// Returns the session IDs that were affected.
func (p *Parser) ParseFile(path string) ([]string, error) {
	offset, err := p.store.GetFileOffset(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Check for truncation
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if offset > fi.Size() {
		offset = 0 // file was truncated, re-parse from start
	}
	if offset == fi.Size() {
		return nil, nil // nothing new
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	info := ExtractSessionInfo(path)

	// Read all events, deduplicating by requestId (last event wins)
	type eventEntry struct {
		event RawEvent
		order int
	}
	byRequestID := make(map[string]eventEntry)
	var noRequestID []RawEvent
	orderCounter := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event RawEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue // skip malformed lines
		}

		// Only process assistant messages with usage data
		if event.Type != "assistant" {
			continue
		}
		if event.Message.Usage.OutputTokens == 0 && event.Message.Usage.InputTokens == 0 &&
			event.Message.Usage.CacheCreationInputTokens == 0 && event.Message.Usage.CacheReadInputTokens == 0 {
			continue
		}

		if event.RequestID != "" {
			byRequestID[event.RequestID] = eventEntry{event: event, order: orderCounter}
		} else {
			noRequestID = append(noRequestID, event)
		}
		orderCounter++
	}

	// Aggregate token usage per session
	type sessionAgg struct {
		model     string
		slug      string
		sessionID string
		timestamp string
		input     int64
		output    int64
		cacheRead int64
		cacheWrite int64
	}
	sessions := make(map[string]*sessionAgg)

	// Collect request-level records for timeline feature
	var requestRecords []store.RequestRecord

	processEvent := func(event RawEvent, requestID string) {
		// Skip events with negative token counts (malformed/malicious data)
		u := event.Message.Usage
		if u.InputTokens < 0 || u.OutputTokens < 0 ||
			u.CacheReadInputTokens < 0 || u.CacheCreationInputTokens < 0 {
			return
		}

		sid := event.SessionID
		if sid == "" {
			sid = info.SessionID
		}
		if sid == "" {
			return
		}

		agg, ok := sessions[sid]
		if !ok {
			agg = &sessionAgg{sessionID: sid}
			sessions[sid] = agg
		}

		if event.Message.Model != "" {
			agg.model = event.Message.Model
		}
		if event.Slug != "" {
			agg.slug = event.Slug
		}
		if event.Timestamp > agg.timestamp {
			agg.timestamp = event.Timestamp
		}

		agg.input += u.InputTokens
		agg.output += u.OutputTokens
		agg.cacheRead += u.CacheReadInputTokens
		agg.cacheWrite += u.CacheCreationInputTokens

		// Store per-request record if we have a requestID
		if requestID != "" {
			usage := calculator.TokenUsage{
				InputTokens:      u.InputTokens,
				OutputTokens:     u.OutputTokens,
				CacheReadTokens:  u.CacheReadInputTokens,
				CacheWriteTokens: u.CacheCreationInputTokens,
			}
			cost := calculator.Calculate(event.Message.Model, usage)
			requestRecords = append(requestRecords, store.RequestRecord{
				RequestID:        requestID,
				SessionID:        sid,
				Timestamp:        event.Timestamp,
				Model:            event.Message.Model,
				InputTokens:      u.InputTokens,
				OutputTokens:     u.OutputTokens,
				CacheReadTokens:  u.CacheReadInputTokens,
				CacheWriteTokens: u.CacheCreationInputTokens,
				Cost:             cost.TotalCost,
			})
		}
	}

	for reqID, entry := range byRequestID {
		processEvent(entry.event, reqID)
	}
	for _, event := range noRequestID {
		processEvent(event, "")
	}

	// Upsert each session
	var affectedIDs []string
	for sid, agg := range sessions {
		usage := calculator.TokenUsage{
			InputTokens:      agg.input,
			OutputTokens:     agg.output,
			CacheReadTokens:  agg.cacheRead,
			CacheWriteTokens: agg.cacheWrite,
		}
		cost := calculator.Calculate(agg.model, usage)

		project := info.Project
		delta := store.SessionDelta{
			ID:              sid,
			Project:         project,
			Slug:            agg.slug,
			Model:           agg.model,
			Timestamp:       agg.timestamp,
			DeltaInput:      agg.input,
			DeltaOutput:     agg.output,
			DeltaCacheRead:  agg.cacheRead,
			DeltaCacheWrite: agg.cacheWrite,
			DeltaCost:       cost.TotalCost,
		}

		if err := p.store.UpsertSession(delta); err != nil {
			log.Printf("Warning: failed to upsert session %s: %v", sid, err)
			continue
		}
		affectedIDs = append(affectedIDs, sid)
	}

	// Upsert per-request records for timeline feature
	for _, rec := range requestRecords {
		if err := p.store.UpsertRequest(rec); err != nil {
			log.Printf("Warning: failed to upsert request %s: %v", rec.RequestID, err)
		}
	}

	// Update file offset to current position
	newOffset := fi.Size()
	if err := p.store.SetFileOffset(path, newOffset); err != nil {
		return affectedIDs, fmt.Errorf("updating offset: %w", err)
	}

	return affectedIDs, nil
}

// ParseFiles parses a specific set of files (used by watcher for incremental updates).
func (p *Parser) ParseFiles(paths []string) ([]string, error) {
	var allSessions []string
	for _, path := range paths {
		sessions, err := p.ParseFile(path)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			continue
		}
		allSessions = append(allSessions, sessions...)
	}
	return allSessions, nil
}
