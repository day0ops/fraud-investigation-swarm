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

	api *kagentclient.APIClientSet
	gw  *kagentclient.GatewayClientSet

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// newKagentInvoker dials grpcTarget (bare host:port, e.g.
// kagent-controller.kagent.svc.cluster.local:8083) for both the control-plane
// API (CreateAgentInstance) and the A2A gateway -- confirmed live (2026-10-02)
// that kagent-enterprise serves both off the same port.
func newKagentInvoker(grpcTarget, namespace, harness, agentTemplate, tokenURL, clientID, clientSecret string) (*kagentInvoker, error) {
	url := "http://" + grpcTarget
	api, err := kagentclient.NewAPI(url)
	if err != nil {
		return nil, fmt.Errorf("create kagent API client: %w", err)
	}
	gw, err := kagentclient.NewGateway(url)
	if err != nil {
		return nil, fmt.Errorf("create kagent gateway client: %w", err)
	}
	return &kagentInvoker{
		namespace:     namespace,
		harness:       harness,
		agentTemplate: agentTemplate,
		tokenURL:      tokenURL,
		clientID:      clientID,
		clientSecret:  clientSecret,
		api:           api,
		gw:            gw,
	}, nil
}

// submit creates one AgentInstance for alertID and sends it task as the
// opening A2A message. Returns the AgentInstance id, used as the case's
// correlation id for chat-span lookups.
func (k *kagentInvoker) submit(ctx context.Context, alertID, task string) (string, error) {
	authCtx, err := k.authContext(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch access token: %w", err)
	}
	createResp, err := k.api.AgentInstance.CreateAgentInstance(authCtx, &apiv1alpha1.CreateAgentInstanceRequest{
		Harness:       &apiv1alpha1.ResourceReference{Namespace: k.namespace, Name: k.harness},
		AgentTemplate: &apiv1alpha1.ResourceReference{Namespace: k.namespace, Name: k.agentTemplate},
		RequestId:     alertID,
	})
	if err != nil {
		return "", fmt.Errorf("create agent instance: %w", err)
	}
	instanceID := createResp.GetAgentInstance().GetId()

	a2aClient, err := k.gw.A2A.ForAgentInstance(authCtx, instanceID)
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
