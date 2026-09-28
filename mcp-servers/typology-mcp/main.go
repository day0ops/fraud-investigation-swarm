// Command typology-mcp is an MCP server matching fraud/AML typologies against
// supplied indicators (or an alert's suspected typology ids) over the shared
// synthetic dataset. All data is fabricated.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "typology-mcp"

type MatchTypologiesInput struct {
	Indicators  []string `json:"indicators,omitempty" jsonschema:"observed indicators to match, e.g. new_device, sanctioned_counterparty"`
	TypologyIDs []string `json:"typology_ids,omitempty" jsonschema:"explicit typology ids to fetch, e.g. from an alert"`
}
type MatchTypologiesOutput struct {
	Typologies []fx.Typology `json:"typologies" jsonschema:"matched typologies"`
}

func matchTypologiesTool(_ context.Context, _ *mcp.CallToolRequest, in MatchTypologiesInput) (*mcp.CallToolResult, MatchTypologiesOutput, error) {
	if len(in.TypologyIDs) > 0 {
		return nil, MatchTypologiesOutput{Typologies: fx.TypologiesByIDs(in.TypologyIDs)}, nil
	}
	return nil, MatchTypologiesOutput{Typologies: fx.MatchTypologies(in.Indicators)}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "match_typologies", Description: "Match fraud/AML typologies from indicators or explicit ids"}, matchTypologiesTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	addr := ":" + envOr("PORT", "9113")
	log.Printf("%s MCP server listening on %s (POST /mcp)", serverName, addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // demo server
		log.Fatalf("server error: %v", err)
	}
}
