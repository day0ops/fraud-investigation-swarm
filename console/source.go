package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// ActorSource yields the live swarm actors for the swarm view.
type ActorSource interface {
	ListActors(ctx context.Context) ([]Actor, error)
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
// api.<ateNamespace>.svc.cluster.local:443` authenticates using the pod's own
// ServiceAccount token with no extra plumbing. The exact JSON field names for
// `get actors` are inferred from the `get workers` shape (no actor existed
// yet to sample directly) -- parsing below is deliberately defensive so an
// unexpected but plausible shape degrades to an empty Template/WorkerPod
// rather than an error; tighten once a live actor can be sampled.
type kubectlAteActorSource struct {
	binPath  string // path to the bundled kubectl-ate binary
	endpoint string // e.g. api.ate-system.svc.cluster.local:443
}

func newKubectlAteActorSource(binPath, endpoint string) *kubectlAteActorSource {
	return &kubectlAteActorSource{binPath: binPath, endpoint: endpoint}
}

type ateMetadata struct {
	Name string `json:"name"`
}

type ateActorStatus struct {
	State string `json:"state"`
}

// ateActorRecord covers the field name candidates a kagent.dev/ate.dev actor
// JSON record may use for its template reference and hosting worker pod --
// see the kubectlAteActorSource doc comment.
type ateActorRecord struct {
	Metadata      ateMetadata    `json:"metadata"`
	ActorTemplate string         `json:"actorTemplate"`
	Template      string         `json:"template"`
	WorkerPod     string         `json:"workerPod"`
	Status        ateActorStatus `json:"status"`
}

type ateActorsResponse struct {
	Actors []ateActorRecord `json:"actors"`
}

func (s *kubectlAteActorSource) ListActors(ctx context.Context) ([]Actor, error) {
	cmd := exec.CommandContext(ctx, s.binPath, "get", "actors", "-A", "--endpoint", s.endpoint, "-o", "json") //nolint:gosec // fixed binary, fixed args
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl-ate get actors: %w: %s", err, stderr.String())
	}
	var resp ateActorsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing kubectl-ate actors output: %w", err)
	}
	actors := make([]Actor, 0, len(resp.Actors))
	for _, a := range resp.Actors {
		template := a.ActorTemplate
		if template == "" {
			template = a.Template
		}
		actors = append(actors, Actor{
			Name:      a.Metadata.Name,
			Template:  template,
			State:     a.Status.State,
			WorkerPod: a.WorkerPod,
		})
	}
	return actors, nil
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
