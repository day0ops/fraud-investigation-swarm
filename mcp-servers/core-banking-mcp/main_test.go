package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetCustomerTool(t *testing.T) {
	_, out, err := getCustomerTool(context.Background(), &mcp.CallToolRequest{}, GetCustomerInput{CustomerID: "CUST-1001"})
	if err != nil {
		t.Fatalf("getCustomerTool() error = %v", err)
	}
	if !out.Found || out.Customer.Name != "Elena Rossi" {
		t.Errorf("got %+v; want found Elena Rossi", out)
	}
}

func TestGetCustomerToolNotFound(t *testing.T) {
	_, out, err := getCustomerTool(context.Background(), &mcp.CallToolRequest{}, GetCustomerInput{CustomerID: "nope"})
	if err != nil {
		t.Fatalf("getCustomerTool() error = %v", err)
	}
	if out.Found {
		t.Error("Found = true; want false for unknown customer")
	}
}

func TestGetTransactionHistoryTool(t *testing.T) {
	_, out, err := getTransactionHistoryTool(context.Background(), &mcp.CallToolRequest{}, GetTransactionHistoryInput{AccountID: "ACC-1001"})
	if err != nil {
		t.Fatalf("getTransactionHistoryTool() error = %v", err)
	}
	if len(out.Transactions) != 3 {
		t.Errorf("got %d txs; want 3", len(out.Transactions))
	}
}
