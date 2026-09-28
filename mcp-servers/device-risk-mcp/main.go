// Command device-risk-mcp is an MCP server exposing device fingerprint and
// IP/geo risk lookups over the shared synthetic dataset. All data is fabricated.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "device-risk-mcp"

type GetDeviceSignalsInput struct {
	CustomerID string `json:"customer_id" jsonschema:"the customer id whose device signals to list"`
}
type GetDeviceSignalsOutput struct {
	Signals []fx.DeviceSignal `json:"signals" jsonschema:"device signals for the customer"`
}

func getDeviceSignalsTool(_ context.Context, _ *mcp.CallToolRequest, in GetDeviceSignalsInput) (*mcp.CallToolResult, GetDeviceSignalsOutput, error) {
	return nil, GetDeviceSignalsOutput{Signals: fx.DeviceSignalsByCustomer(in.CustomerID)}, nil
}

type GetIPGeoInput struct {
	IP string `json:"ip" jsonschema:"the IP address to geolocate and risk-score"`
}
type GetIPGeoOutput struct {
	Found bool     `json:"found" jsonschema:"whether the IP is known"`
	IPGeo fx.IPGeo `json:"ip_geo,omitempty" jsonschema:"geo and risk score if known"`
}

func getIPGeoTool(_ context.Context, _ *mcp.CallToolRequest, in GetIPGeoInput) (*mcp.CallToolResult, GetIPGeoOutput, error) {
	g, ok := fx.IPGeoByIP(in.IP)
	return nil, GetIPGeoOutput{Found: ok, IPGeo: g}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "get_device_signals", Description: "List device fingerprint signals for a customer"}, getDeviceSignalsTool)
	mcp.AddTool(server, &mcp.Tool{Name: "get_ip_geo", Description: "Geolocate and risk-score an IP address"}, getIPGeoTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	addr := ":" + envOr("PORT", "9112")
	log.Printf("%s MCP server listening on %s (POST /mcp)", serverName, addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // demo server
		log.Fatalf("server error: %v", err)
	}
}
