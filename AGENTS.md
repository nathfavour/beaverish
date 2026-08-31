# Beaverish - Autonomous Agent & Orchestrator Guide

## 1. System Orchestration & Directives
1. You are an autonomous software engineering and trading agent operating within the **Beaverish** ecosystem.
2. The primary domain of Beaverish is high-throughput binary event contract execution, spread/parity arbitrage evaluation, settlement redemption, and embedded Model Context Protocol (MCP) server stdio streaming on the Somnia EVM network (Chain ID: 50312).
3. For architectural context and domain structures, reference [ARCHITECTURE.md](ARCHITECTURE.md). For MCP specs, reference [docs/MCP.md](docs/MCP.md) or [`.agents/skills/mcp/SKILL.md`](.agents/skills/mcp/SKILL.md).

---

## 2. Agent Execution Lifecycle & Skills Directory

Before proposing architectural shifts or tool additions, consult the skill catalog in `.agents/skills/`:
- **`mcp`**: [`.agents/skills/mcp/SKILL.md`](.agents/skills/mcp/SKILL.md) — Connect Claude Code, Codex, Antigravity, and Cursor to Beaverish via JSON-RPC stdio.
- **`wallet-gen`**: [`.agents/skills/wallet-gen/SKILL.md`](.agents/skills/wallet-gen/SKILL.md) — Fast HD wallet derivation and key generation using `hdwallet-cli`.
- **`dogfooding`**: [`.agents/skills/dogfooding/SKILL.md`](.agents/skills/dogfooding/SKILL.md) — Live testing, dry-run simulation, and JSON-RPC dogfooding instructions.
- **`somnia-dex`**: [`.agents/skills/somnia-dex/SKILL.md`](.agents/skills/somnia-dex/SKILL.md) — DreamDEX contract interaction, orderbook structures, and allowance mechanics.
- **`beaverish`**: [`.agents/skills/beaverish/SKILL.md`](.agents/skills/beaverish/SKILL.md) — Core daemon architecture, signal evaluators, and state machines.

---

## 3. 🏗️ Engineering & Safety Directives

### 🚫 Strict MCP Stdio Isolation
- When running in `--mcp` mode, `os.Stdout` is strictly reserved for newline-delimited JSON-RPC 2.0 messages.
- ALL operational logs, diagnostic traces, and telemetry MUST route strictly to `os.Stderr`. Writing unstructured text to `stdout` breaks host agent communication.

### 🛡️ Nonce & Allowance Safety
- Always use the thread-safe `NonceManager` for on-chain state transitions.
- Ensure pre-flight ERC-20 allowances are verified before dispatching high-frequency trade workers.

### ⚡ Source Control & Commit Standards
- Commit clean, descriptive messages without co-author noise.
- Keep tests passing with race detection enabled: `go test -v -race ./...`.

---

## 4. Dogfooding & Local Verification
- Test MCP tools locally over stdio:
  ```bash
  printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n' | beaverish --mcp
  ```
- Generate disposable test wallets using `hdwallet-cli` (see `.agents/skills/wallet-gen/SKILL.md`).
