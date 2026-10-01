// kagentInvoker starts fraud investigations against kagent Agent Substrate:
// one AgentInstance per alert (CreateAgentInstance, idempotent on the alert id
// as the request_id), then an A2A SendMessage carrying the alert brief as the
// opening task. This deployment enforces real Keycloak OIDC on the control
// plane, so every call carries a client-credentials bearer token.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	kagentclient "github.com/kagent-dev/kagent/go/api/client"
	apiv1alpha1 "github.com/kagent-dev/kagent/go/api/gen/kagent/api/v1alpha1"
	"google.golang.org/grpc/metadata"
)

type kagentInvoker struct {
	namespace     string
	harness       string
	agentTemplate string
	tokenURL      string
	clientID      string
	clientSecret  string

	instances *kagentclient.AgentInstanceClient
	a2a       *kagentclient.A2AClient

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func newKagentInvoker(grpcTarget, namespace, harness, agentTemplate, tokenURL, clientID, clientSecret string) *kagentInvoker {
	base := kagentclient.NewBaseClient("", kagentclient.WithGRPCTarget(grpcTarget))
	return &kagentInvoker{
		namespace:     namespace,
		harness:       harness,
		agentTemplate: agentTemplate,
		tokenURL:      tokenURL,
		clientID:      clientID,
		clientSecret:  clientSecret,
		instances:     kagentclient.NewAgentInstanceClient(base),
		a2a:           kagentclient.NewA2AClient(base),
	}
}

// submit creates one AgentInstance for alertID and sends it task as the
// opening A2A message. Returns the AgentInstance id, used as the case's
// correlation id for chat-span lookups.
func (k *kagentInvoker) submit(ctx context.Context, alertID, task string) (string, error) {
	authCtx, err := k.authContext(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch access token: %w", err)
	}
	createResp, err := k.instances.CreateAgentInstance(authCtx, &apiv1alpha1.CreateAgentInstanceRequest{
		Namespace:     k.namespace,
		Harness:       k.harness,
		AgentTemplate: k.agentTemplate,
		RequestId:     alertID,
	})
	if err != nil {
		return "", fmt.Errorf("create agent instance: %w", err)
	}
	instanceID := createResp.GetAgentInstance().GetId()

	a2aClient, err := k.a2a.ForAgentInstance(authCtx, k.namespace, instanceID)
	if err != nil {
		return "", fmt.Errorf("a2a client for agent instance %s: %w", instanceID, err)
	}
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(task))
	if _, err := a2aClient.SendMessage(authCtx, &a2a.SendMessageRequest{Message: msg}); err != nil {
		return "", fmt.Errorf("send message to agent instance %s: %w", instanceID, err)
	}
	return instanceID, nil
}

func (k *kagentInvoker) authContext(ctx context.Context) (context.Context, error) {
	token, err := k.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), nil
}

// accessToken fetches (and caches, refreshing 30s ahead of expiry) a
// client-credentials token from Keycloak for the confidential client this
// console runs as.
func (k *kagentInvoker) accessToken(ctx context.Context) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.token != "" && time.Now().Before(k.tokenExpiry) {
		return k.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {k.clientID},
		"client_secret": {k.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.tokenURL, bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("keycloak token request: status %d", resp.StatusCode)
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode keycloak token response: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("keycloak token response had no access_token")
	}
	k.token = out.AccessToken
	k.tokenExpiry = time.Now().Add(time.Duration(out.ExpiresIn)*time.Second - 30*time.Second)
	return k.token, nil
}
