package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMatchTypologiesByIndicators(t *testing.T) {
	_, out, err := matchTypologiesTool(context.Background(), &mcp.CallToolRequest{}, MatchTypologiesInput{Indicators: []string{"new_device", "sanctioned_counterparty"}})
	if err != nil {
		t.Fatalf("matchTypologiesTool() error = %v", err)
	}
	ids := map[string]bool{}
	for _, ty := range out.Typologies {
		ids[ty.ID] = true
	}
	if !ids["TYP-ATO"] || !ids["TYP-SANCTIONS"] {
		t.Errorf("got %v; want TYP-ATO and TYP-SANCTIONS", ids)
	}
}

func TestMatchTypologiesEmpty(t *testing.T) {
	_, out, err := matchTypologiesTool(context.Background(), &mcp.CallToolRequest{}, MatchTypologiesInput{Indicators: []string{"unrelated_indicator"}})
	if err != nil {
		t.Fatalf("matchTypologiesTool() error = %v", err)
	}
	if len(out.Typologies) != 0 {
		t.Errorf("got %d; want 0 for an unrelated indicator", len(out.Typologies))
	}
}
