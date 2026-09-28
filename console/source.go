package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// ActorSource yields the live swarm actors and worker-pool size for the swarm view.
type ActorSource interface {
	ListActors(ctx context.Context) ([]Actor, error)
	WorkerPoolSize(ctx context.Context) (int, error)
}

// SpanSource yields a case's orchestration spans for the graph, keyed by the
// kagent ConversationId/TaskId the case was correlated under (not the raw
// alertID -- see server.go's alertCorrelations map).
type SpanSource interface {
	CaseSpans(ctx context.Context, correlationID string) ([]SpanNode, error)
}

// kubectlAteActorSource shells out to the kubectl-ate CLI (bundled in this
// image) rather than hand-writing an ate-api gRPC client. Live-confirmed:
// from inside the cluster, `kubectl-ate get workers/actors --endpoint
// api.<ateNamespace>.svc.cluster.local:443` authenticates as the `ate-client`
// ServiceAccount via a TokenRequest -- this feature's own ServiceAccount must
// be granted `serviceaccounts/token: create` on `ate-client` in ate-system
// (see the fraud-ops-console feature's RBAC). `get actors` records do not
// carry a worker-pod field at all (only `get workers` does), so worker-pod
// occupancy is fetched separately via WorkerPoolSize.
type kubectlAteActorSource struct {
	binPath   string // path to the bundled kubectl-ate binary
	endpoint  string // e.g. api.ate-system.svc.cluster.local:443
	namespace string // kagent namespace the fraud-workers WorkerPool lives in
}

func newKubectlAteActorSource(binPath, endpoint, namespace string) *kubectlAteActorSource {
	return &kubectlAteActorSource{binPath: binPath, endpoint: endpoint, namespace: namespace}
}

type ateMetadata struct {
	Name string `json:"name"`
}

type ateActorStatus struct {
	State string `json:"state"`
}

// ateActorTemplateRef is the actorTemplate object on an actor record --
// live-confirmed shape: {"atespace": "...", "name": "<template>-<harness>-<revision>"}.
type ateActorTemplateRef struct {
	Name string `json:"name"`
}

// ateActorRecord is the live-confirmed shape of one `kubectl-ate get actors`
// entry. metadata.name is the actor's own unique id (a UUID), not a template
// name -- actorTemplate.name is the human-readable template/harness reference.
type ateActorRecord struct {
	Metadata      ateMetadata         `json:"metadata"`
	ActorTemplate ateActorTemplateRef `json:"actorTemplate"`
	Status        ateActorStatus      `json:"status"`
}

type ateActorsResponse struct {
	Actors []ateActorRecord `json:"actors"`
}

type ateWorkersResponse struct {
	Workers []json.RawMessage `json:"workers"`
}

func (s *kubectlAteActorSource) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, s.binPath, append(args, "--endpoint", s.endpoint, "-o", "json")...) //nolint:gosec // fixed binary, fixed args
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl-ate %v: %w: %s", args, err, stderr.String())
	}
	return out, nil
}

// ateStateToActorState normalizes ate's ACTOR_STATE_RUNNING-style enum values
// to the lowercase form the console SPA's state-color map expects.
func ateStateToActorState(s string) string {
	s = strings.TrimPrefix(s, "ACTOR_STATE_")
	return strings.ToLower(s)
}

func (s *kubectlAteActorSource) ListActors(ctx context.Context) ([]Actor, error) {
	out, err := s.run(ctx, "get", "actors", "-A")
	if err != nil {
		return nil, err
	}
	var resp ateActorsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing kubectl-ate actors output: %w", err)
	}
	actors := make([]Actor, 0, len(resp.Actors))
	for _, a := range resp.Actors {
		actors = append(actors, Actor{
			Name:     a.Metadata.Name,
			Template: a.ActorTemplate.Name,
			State:    ateStateToActorState(a.Status.State),
		})
	}
	return actors, nil
}

// WorkerPoolSize returns how many worker pods currently exist in namespace --
// actor records carry no worker-pod field, so this is queried separately.
func (s *kubectlAteActorSource) WorkerPoolSize(ctx context.Context) (int, error) {
	out, err := s.run(ctx, "get", "workers", "-n", s.namespace)
	if err != nil {
		return 0, err
	}
	var resp ateWorkersResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, fmt.Errorf("parsing kubectl-ate workers output: %w", err)
	}
	return len(resp.Workers), nil
}

// clickhouseSpanSource reads a case's parent->child agent spans from
// kagent.kagent_chat_spans. Live-confirmed schema: SpanId/ParentSpanId give
// the orchestration tree and ServiceName carries the agent/template name, but
// there is no native alertId column -- callers must supply the
// ConversationId/TaskId this case was correlated under (captured by the
// server at trigger time; see server.go's alertCorrelations map).
type clickhouseSpanSource struct {
	db *sql.DB
}

func newClickhouseSpanSource(dsn string) (*clickhouseSpanSource, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening clickhouse connection: %w", err)
	}
	return &clickhouseSpanSource{db: db}, nil
}

// CaseSpans looks up spans by the correlation id (ConversationId or TaskId)
// this case was submitted under, not by alertID directly.
func (s *clickhouseSpanSource) CaseSpans(ctx context.Context, correlationID string) ([]SpanNode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT SpanId, ParentSpanId, ServiceName, Timestamp
		 FROM kagent.kagent_chat_spans
		 WHERE ConversationId = ? OR TaskId = ?
		 ORDER BY Timestamp ASC`,
		correlationID, correlationID)
	if err != nil {
		return nil, fmt.Errorf("querying chat spans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var nodes []SpanNode
	for rows.Next() {
		var n SpanNode
		var startedAt any
		if err := rows.Scan(&n.ID, &n.ParentID, &n.Agent, &startedAt); err != nil {
			return nil, fmt.Errorf("scanning chat span row: %w", err)
		}
		n.StartedAt = fmt.Sprintf("%v", startedAt)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
