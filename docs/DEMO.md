# Demo run of show

A presenter script for the fraud/AML investigation swarm on kagent Agent Substrate. The story: every fraud alert gets its own isolated, gVisor-sandboxed investigation team (a lead investigator orchestrating five specialists), multiplexed onto a small pool of Substrate workers.

## Prerequisites

- The `fraud-investigation-swarm` use case deployed on the target cluster (see `agentic-field-kit`'s `config/usecases/single-cluster/agentic/fraud-investigation-swarm/`).
- The Fraud Ops Console reachable at its configured hostname.
- All four mock MCP servers and the six-agent swarm healthy (`kubectl get harness,agenttemplate,mcpserver,remotemcpserver -n kagent`).

## The script

1. **Cold open.** Open the console. Point out the quiet queue: 0 active investigations, N worker pods idle. This is the empty state - nothing is running yet.
2. **The hero case.** Click "Submit hero case" (`ALERT-1001`: an account takeover with two large wires to a sanctioned payee, minutes after a new-device login from a high-risk geo). Open the case. Walk through the lead investigator delegating to all five specialists in turn, then the ESCALATE disposition citing the sanctioned-payee hit and the account-takeover signals.
3. **The burst.** Click "Simulate fraud campaign". Watch tiles bloom across the grid as investigations spin up - the density header climbs to roughly 10 active investigations while the worker-pod count stays flat at 2-3. This is the headline Substrate claim: far more concurrent work than there are pods.
4. **Isolation proof.** Show that a specialist (e.g. `tx-analyst`) is bound to exactly one MCP server (`core-banking-mcp`) and nothing else - it has no path to sanctions or device-risk data. Point to the negative test result confirming that denial.
5. **The settle.** Let the burst finish. Tiles grey out and vanish as investigations complete; the worker-pod count never moved. Deliver the punchline: dozens of isolated investigations, a handful of pods, zero cross-contamination.

## Talking points

- Every agent is a real, live LLM call (not scripted) - only the bank data is synthetic.
- Specialists run on a cheaper/faster model; the lead investigator and narrator run on a stronger one, keeping a full campaign affordable and repeatable.
- The console reads the live swarm two ways: `kubectl-ate` for actor/worker state, ClickHouse (`kagent_chat_spans`) for the per-case orchestration graph.
