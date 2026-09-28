package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestScreenSanctionsHit(t *testing.T) {
	_, out, err := screenSanctionsTool(context.Background(), &mcp.CallToolRequest{}, ScreenInput{Name: "Zarwan Holdings LLC"})
	if err != nil {
		t.Fatalf("screenSanctionsTool() error = %v", err)
	}
	if !out.Hit || len(out.Matches) != 1 || out.Matches[0].List != "OFAC-SDN" {
		t.Errorf("got %+v; want one OFAC-SDN hit", out)
	}
}

func TestScreenSanctionsClean(t *testing.T) {
	_, out, err := screenSanctionsTool(context.Background(), &mcp.CallToolRequest{}, ScreenInput{Name: "Supermercato Centrale"})
	if err != nil {
		t.Fatalf("screenSanctionsTool() error = %v", err)
	}
	if out.Hit {
		t.Error("Hit = true; want false for a clean counterparty")
	}
}

func TestScreenPEPHit(t *testing.T) {
	_, out, err := screenPEPTool(context.Background(), &mcp.CallToolRequest{}, ScreenInput{Name: "David Okoro"})
	if err != nil {
		t.Fatalf("screenPEPTool() error = %v", err)
	}
	if !out.Hit {
		t.Error("Hit = false; want true for a PEP")
	}
}
