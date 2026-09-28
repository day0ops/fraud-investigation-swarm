# Fraud Investigation Swarm

![Fraud Investigation Swarm](images/banner.png)

[![ci](https://github.com/day0ops/fraud-investigation-swarm/actions/workflows/ci.yml/badge.svg)](https://github.com/day0ops/fraud-investigation-swarm/actions/workflows/ci.yml)

A demo of a financial-crime investigation swarm running on kagent Agent Substrate: every fraud alert spins up its own isolated, gVisor-sandboxed investigation team (a lead investigator orchestrating five specialists), multiplexed onto a small pool of Substrate workers.

This repository holds the application tier: the mock MCP data servers, the alert driver, the Fraud Ops Console, and the shared synthetic dataset. The deployment glue that runs it on a cluster lives in `agentic-field-kit`.

## Components

| Path | Purpose |
|---|---|
| `fixtures/` | Shared synthetic dataset (customers, transactions, watchlist hits, device signals, typologies, alerts) |
| `mcp-servers/core-banking-mcp/` | MCP server: customer, account, and transaction-history lookups |
| `mcp-servers/watchlist-mcp/` | MCP server: sanctions and PEP screening |
| `mcp-servers/device-risk-mcp/` | MCP server: device fingerprint and IP/geo risk |
| `mcp-servers/typology-mcp/` | MCP server: fraud/AML typology matching |
| `alert-driver/` | CLI to trigger single or burst fraud investigations |
| `console/` | Fraud Ops Console: Go backend (embeds the built SPA) + Vite frontend |

See [docs/DEMO.md](docs/DEMO.md) for the presenter run of show.

## MCP tools

| Server | Port | Tools |
|---|---|---|
| core-banking-mcp | 9110 | `get_customer`, `get_account`, `get_transaction_history` |
| watchlist-mcp | 9111 | `screen_sanctions`, `screen_pep` |
| device-risk-mcp | 9112 | `get_device_signals`, `get_ip_geo` |
| typology-mcp | 9113 | `match_typologies` |

## Development

```bash
go work sync
go build ./...
go test ./...
```

## License

Apache-2.0. See [LICENSE](LICENSE).
