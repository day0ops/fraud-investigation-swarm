// Command watchlist-mcp is an MCP server exposing sanctions and PEP screening
// over the shared synthetic dataset. All data is fabricated.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "watchlist-mcp"

type ScreenInput struct {
	Name string `json:"name" jsonschema:"the counterparty or customer name to screen"`
}
type ScreenOutput struct {
	Hit     bool                `json:"hit" jsonschema:"whether any watchlist entry matched"`
	Matches []fx.WatchlistEntry `json:"matches" jsonschema:"matched watchlist entries"`
}

func screenSanctionsTool(_ context.Context, _ *mcp.CallToolRequest, in ScreenInput) (*mcp.CallToolResult, ScreenOutput, error) {
	m := fx.ScreenSanctions(in.Name)
	return nil, ScreenOutput{Hit: len(m) > 0, Matches: m}, nil
}

func screenPEPTool(_ context.Context, _ *mcp.CallToolRequest, in ScreenInput) (*mcp.CallToolResult, ScreenOutput, error) {
	m := fx.ScreenPEP(in.Name)
	return nil, ScreenOutput{Hit: len(m) > 0, Matches: m}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "screen_sanctions", Description: "Screen a name against sanctions lists (OFAC/EU/UN)"}, screenSanctionsTool)
	mcp.AddTool(server, &mcp.Tool{Name: "screen_pep", Description: "Screen a name against politically-exposed-person lists"}, screenPEPTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	addr := ":" + envOr("PORT", "9111")
	log.Printf("%s MCP server listening on %s (POST /mcp)", serverName, addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // demo server
		log.Fatalf("server error: %v", err)
	}
}
