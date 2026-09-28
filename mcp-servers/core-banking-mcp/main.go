// Command core-banking-mcp is an MCP server exposing read-only customer,
// account, and transaction-history lookups over the shared synthetic dataset.
// All data is fabricated. No auth is enforced here; the gateway is the
// enforcement point, matching the sibling day0ops demos.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	fx "github.com/day0ops/fraud-investigation-swarm/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "core-banking-mcp"

type GetCustomerInput struct {
	CustomerID string `json:"customer_id" jsonschema:"the customer id to look up, e.g. CUST-1001"`
}
type GetCustomerOutput struct {
	Found    bool        `json:"found" jsonschema:"whether the customer exists"`
	Customer fx.Customer `json:"customer,omitempty" jsonschema:"the customer record if found"`
}

func getCustomerTool(_ context.Context, _ *mcp.CallToolRequest, in GetCustomerInput) (*mcp.CallToolResult, GetCustomerOutput, error) {
	c, ok := fx.CustomerByID(in.CustomerID)
	return nil, GetCustomerOutput{Found: ok, Customer: c}, nil
}

type GetAccountInput struct {
	AccountID string `json:"account_id" jsonschema:"the account id to look up, e.g. ACC-1001"`
}
type GetAccountOutput struct {
	Found   bool       `json:"found" jsonschema:"whether the account exists"`
	Account fx.Account `json:"account,omitempty" jsonschema:"the account record if found"`
}

func getAccountTool(_ context.Context, _ *mcp.CallToolRequest, in GetAccountInput) (*mcp.CallToolResult, GetAccountOutput, error) {
	a, ok := fx.AccountByID(in.AccountID)
	return nil, GetAccountOutput{Found: ok, Account: a}, nil
}

type GetTransactionHistoryInput struct {
	AccountID string `json:"account_id" jsonschema:"the account id whose transactions to list"`
}
type GetTransactionHistoryOutput struct {
	Transactions []fx.Transaction `json:"transactions" jsonschema:"transactions for the account, newest first is not guaranteed"`
}

func getTransactionHistoryTool(_ context.Context, _ *mcp.CallToolRequest, in GetTransactionHistoryInput) (*mcp.CallToolResult, GetTransactionHistoryOutput, error) {
	return nil, GetTransactionHistoryOutput{Transactions: fx.TransactionsByAccount(in.AccountID)}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "get_customer", Description: "Look up a customer by id"}, getCustomerTool)
	mcp.AddTool(server, &mcp.Tool{Name: "get_account", Description: "Look up an account by id"}, getAccountTool)
	mcp.AddTool(server, &mcp.Tool{Name: "get_transaction_history", Description: "List transactions for an account"}, getTransactionHistoryTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	addr := ":" + envOr("PORT", "9110")
	log.Printf("%s MCP server listening on %s (POST /mcp)", serverName, addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // demo server
		log.Fatalf("server error: %v", err)
	}
}
