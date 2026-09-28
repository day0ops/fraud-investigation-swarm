// Command console serves the Fraud Ops Console: a JSON API over the live
// swarm plus the embedded Vite SPA. Actor state comes from an ActorSource,
// case graphs from a SpanSource, and the trigger endpoint starts
// investigations via the same recipe the alert driver uses (LEAD_ENDPOINT).
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
)

//go:embed all:web/dist
var webDist embed.FS

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

type server struct {
	actors       ActorSource
	spans        SpanSource
	leadEndpoint string

	mu                sync.Mutex
	alertCorrelations map[string]string // alertID -> ConversationId/TaskId, captured at submit time
}

func (s *server) handleActors(w http.ResponseWriter, r *http.Request) {
	a, err := s.actors.ListActors(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, SummarizeSwarm(a))
}

func (s *server) handleCaseSpans(w http.ResponseWriter, r *http.Request) {
	alertID := r.PathValue("id")
	s.mu.Lock()
	correlationID, ok := s.alertCorrelations[alertID]
	s.mu.Unlock()
	if !ok {
		// Not yet correlated (or submitted before this server started) -- fall
		// back to the alertID itself in case the invocation recipe ends up
		// setting the correlation id to it directly.
		correlationID = alertID
	}
	spans, err := s.spans.CaseSpans(r.Context(), correlationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, BuildCaseGraph(alertID, spans))
}

type alertRequest struct {
	Mode    string `json:"mode"` // "single" | "burst"
	AlertID string `json:"alertId"`
	Count   int    `json:"count"`
}

func (s *server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	var req alertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var ids []string
	switch req.Mode {
	case "single":
		if _, ok := fx.AlertByID(req.AlertID); ok {
			ids = []string{req.AlertID}
		}
	case "burst":
		all := fx.AllAlerts()
		for i := 0; i < req.Count && len(all) > 0; i++ {
			ids = append(ids, all[i%len(all)].ID)
		}
	}
	if len(ids) == 0 {
		http.Error(w, "no alerts selected", http.StatusBadRequest)
		return
	}
	for _, id := range ids {
		if err := s.submit(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
	writeJSON(w, map[string]any{"submitted": len(ids)})
}

// submit starts one investigation for alertID via the lead-investigator
// invocation recipe. Correlation-id capture is provisional: the exact
// recipe/response shape is confirmed live in the deployment plan (Plan 2
// Task 5, blocked pending published images); until then this records
// alertID -> alertID so handleCaseSpans has a deterministic fallback.
func (s *server) submit(ctx context.Context, alertID string) error {
	a, _ := fx.AlertByID(alertID)
	body, _ := json.Marshal(map[string]any{
		"alertId": a.ID, "customerId": a.CustomerID, "accountId": a.AccountID,
		"type": a.Type, "triggerReason": a.TriggerReason,
	})
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.leadEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	correlationID := alertID
	var out struct {
		ConversationID string `json:"conversationId"`
		TaskID         string `json:"taskId"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) == nil {
		if out.ConversationID != "" {
			correlationID = out.ConversationID
		} else if out.TaskID != "" {
			correlationID = out.TaskID
		}
	}
	s.mu.Lock()
	s.alertCorrelations[alertID] = correlationID
	s.mu.Unlock()
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	spanSource, err := newClickhouseSpanSource(envOr("CLICKHOUSE_DSN", ""))
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	s := &server{
		actors: newKubectlAteActorSource(
			envOr("KUBECTL_ATE_PATH", "/usr/local/bin/kubectl-ate"),
			envOr("ATE_API_ENDPOINT", "api.ate-system.svc.cluster.local:443"),
		),
		spans:             spanSource,
		leadEndpoint:      envOr("LEAD_ENDPOINT", ""),
		alertCorrelations: map[string]string{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/actors", s.handleActors)
	mux.HandleFunc("GET /api/case/{id}/spans", s.handleCaseSpans)
	mux.HandleFunc("POST /api/alerts", s.handleAlerts)

	dist, _ := fs.Sub(webDist, "web/dist")
	mux.Handle("/", http.FileServer(http.FS(dist)))

	addr := ":" + envOr("PORT", "8090")
	log.Printf("fraud-ops-console listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // demo server
		log.Fatalf("server error: %v", err)
	}
}
