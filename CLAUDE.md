# CLAUDE.md

Guidance for Claude Code in this repository.

## Overview

Application tier for the fraud/AML investigation swarm demo on kagent Agent Substrate. Go monorepo (`go.work`): four MCP data servers, an alert driver, a Fraud Ops Console (Go backend + Vite SPA), and a shared synthetic `fixtures` module. Deployment glue lives in the separate `agentic-field-kit` repo.

## Conventions

- Go 1.27.0. MCP servers use `github.com/modelcontextprotocol/go-sdk` served at `POST /mcp`.
- Each component is its own module under `github.com/day0ops/fraud-investigation-swarm/`; add it to `go.work`.
- Shared data lives in `fixtures/` and is imported by servers and the driver; keep it the single source of truth (DRY).
- All data is synthetic. No real personal or financial data.

## Commands

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

## Style

- No em dash characters in prose; use a plain hyphen or restructure.
