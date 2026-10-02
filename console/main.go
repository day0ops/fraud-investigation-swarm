// Command console serves the Fraud Ops Console: a JSON API over the live
// swarm plus the embedded Vite SPA. Actor state comes from an ActorSource,
// case graphs from a SpanSource, and the trigger endpoint starts
// investigations via kagentInvoker (CreateAgentInstance + A2A SendMessage).
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

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
	actors  ActorSource
	spans   SpanSource
	invoker *kagentInvoker

	mu                sync.Mutex
	alertCorrelations map[string]string // alertID -> AgentInstance id, captured at submit time
	burstRun          atomic.Uint64
}

// nextBurstRun returns a fresh run number, used to make each "Simulate fraud
// campaign" click create genuinely new AgentInstances instead of idempotently
// reusing earlier ones (CreateAgentInstance's idempotency is keyed on the
// request id handed to it; see handleAlerts).
func (s *server) nextBurstRun() uint64 {
	return s.burstRun.Add(1)
}

func (s *server) handleActors(w http.ResponseWriter, r *http.Request) {
	a, err := s.actors.ListActors(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	workerPods, err := s.actors.WorkerPoolSize(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, SummarizeSwarm(a, workerPods))
}

func (s *server) handleCaseSpans(w http.ResponseWriter, r *http.Request) {
	alertID := r.PathValue("id")
	s.mu.Lock()
	correlationID, ok := s.alertCorrelations[alertID]
	s.mu.Unlock()
	if !ok {
		// Not yet correlated (or submitted before this server started) -- fall
		// back to the alertID itself.
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

	// Single mode keeps using the raw alertID as the request id, so repeat
	// clicks of "Submit hero case" idempotently resume the same investigation
	// rather than spawning duplicates. Burst mode instead stamps each call
	// with this run's own number: cycling past the fixture count (or
	// re-running the campaign) must spin up fresh AgentInstances, not
	// idempotently return earlier ones, or "count" would never actually
	// produce more than len(fixtures) concurrent investigations.
	var runPrefix string
	if req.Mode == "burst" {
		runPrefix = fmt.Sprintf("run%d-", s.nextBurstRun())
	}

	// Fired as fast as the handler can loop -- no client-side pacing. Substrate's
	// atenet-router already queues ("parks") resume attempts under worker-pool
	// saturation instead of failing fast, so a naive burst is the honest test.
	// One alert's failure doesn't abort the rest -- a burst is explicitly a
	// resilience-under-load test, not an all-or-nothing submission.
	submitted := 0
	var errs []string
	for i, id := range ids {
		requestID := id
		if runPrefix != "" {
			requestID = fmt.Sprintf("%s%s-%d", runPrefix, id, i)
		}
		if err := s.submit(r.Context(), id, requestID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
			continue
		}
		submitted++
	}
	resp := map[string]any{"submitted": submitted, "requested": len(ids)}
	if len(errs) > 0 {
		resp["errors"] = errs
	}
	writeJSON(w, resp)
}

func (s *server) submit(ctx context.Context, alertID, requestID string) error {
	a, ok := fx.AlertByID(alertID)
	if !ok {
		return fmt.Errorf("unknown alert %s", alertID)
	}
	instanceID, err := s.invoker.submit(ctx, requestID, buildTask(a))
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.alertCorrelations[alertID] = instanceID
	s.mu.Unlock()
	return nil
}

// buildTask renders an alert as the natural-language opening task handed to
// fraud-lead-investigator -- it has no knowledge of the case beyond this.
func buildTask(a fx.Alert) string {
	return fmt.Sprintf(
		"New fraud alert %s for customer %s (account %s).\nAlert type: %s.\nTrigger reason: %s.\n\n"+
			"Investigate this case using your specialists, then give me a disposition (CLEAR or ESCALATE) "+
			"with the supporting evidence and a written case narrative.",
		a.ID, a.CustomerID, a.AccountID, a.Type, a.TriggerReason,
	)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s is required", k)
	}
	return v
}

func main() {
	spanSource, err := newClickhouseSpanSource(envOr("CLICKHOUSE_DSN", ""))
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	namespace := envOr("ATE_NAMESPACE", "kagent")
	invoker, err := newKagentInvoker(
		envOr("KAGENT_GRPC_TARGET", "kagent-controller."+namespace+".svc.cluster.local:8083"),
		namespace,
		envOr("FRAUD_SWARM_HARNESS", "fraud-swarm"),
		envOr("FRAUD_LEAD_AGENT_TEMPLATE", "fraud-lead-investigator"),
		mustEnv("KEYCLOAK_TOKEN_URL"),
		mustEnv("KEYCLOAK_CLIENT_ID"),
		mustEnv("KEYCLOAK_CLIENT_SECRET"),
	)
	if err != nil {
		log.Fatalf("kagent invoker: %v", err)
	}
	s := &server{
		actors: newKubectlAteActorSource(
			envOr("KUBECTL_ATE_PATH", "/usr/local/bin/kubectl-ate"),
			envOr("ATE_API_ENDPOINT", "api.ate-system.svc.cluster.local:443"),
			namespace,
			envOr("WORKER_POOL_NAME", "fraud-workers"),
		),
		spans:             spanSource,
		invoker:           invoker,
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
