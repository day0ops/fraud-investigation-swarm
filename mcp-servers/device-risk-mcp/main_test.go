package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetDeviceSignalsTool(t *testing.T) {
	_, out, err := getDeviceSignalsTool(context.Background(), &mcp.CallToolRequest{}, GetDeviceSignalsInput{CustomerID: "CUST-1001"})
	if err != nil {
		t.Fatalf("getDeviceSignalsTool() error = %v", err)
	}
	if len(out.Signals) != 1 || !out.Signals[0].IsNewDevice {
		t.Errorf("got %+v; want one new-device signal", out)
	}
}

func TestGetIPGeoTool(t *testing.T) {
	_, out, err := getIPGeoTool(context.Background(), &mcp.CallToolRequest{}, GetIPGeoInput{IP: "185.220.101.47"})
	if err != nil {
		t.Fatalf("getIPGeoTool() error = %v", err)
	}
	if !out.Found || out.IPGeo.RiskScore != 92 {
		t.Errorf("got %+v; want found risk 92", out)
	}
}
