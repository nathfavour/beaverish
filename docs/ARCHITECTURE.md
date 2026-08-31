## 1. System Overview

Beaverish is a high-throughput, protocol-agnostic daemon written in Go for executing, managing, and settling high-frequency binary event contracts and order-book prediction markets.

While designed with a modular driver abstraction that decouples execution logic from specific blockchain platforms, the primary reference implementation targets DreamDEX Event Contracts deployed on the high-throughput Somnia EVM network (Chain ID: 50312).

The system operates in dual execution modes:
1. Headless Autonomous Loop: A background daemon that evaluates deterministic statistical arbitrage, order-book parity, and latency signals to automatically place, track, and settle binary outcome positions.
2. Embedded Model Context Protocol (MCP) Server: A standard POSIX stdio JSON-RPC interface allowing external AI runtimes and automated orchestrators to query market state, assess risk, and dispatch signed transactions.

---

## 2. Core Architectural Principles

- Zero External SaaS Dependencies: Direct communication with EVM nodes via JSON-RPC/WebSockets and standard POSIX system interfaces. No proprietary cloud middleware or centralized API keys.
- Engine-Driver Decoupling: Complete isolation between core trading/settlement state machines and protocol-specific ABI interactions.
- Deterministic Execution & Concurrency: Goroutine-based event streaming, atomic in-memory nonce management, and non-blocking worker pools.
- XDG Directory Compliance: Strict adherence to OS configuration and state persistence standards.
- Autonomous Lifecycle Sweeping: Guaranteed round-trip trade settlement and balance realization with automated token approval and redemption loops.

---

## 3. High-Level System Topology

```
+-----------------------------------------------------------------------------+
|                         System Entrypoints & Triggers                       |
|                                                                             |
|   +--------------------------+                   +-----------------------+  |
|   |   AI Host / Orchestrator |                   | CLI Operator / Daemon |  |
|   |  (Cursor, Claude, Local) |                   | (systemd / Docker /   |  |
|   +------------+-------------+                   |  Interactive Shell)   |  |
|                |                                 +-----------+-----------+  |
|                | stdio (JSON-RPC 2.0)                        |              |
|                v                                             v              |
|   +---------------------------------------------------------------------+   |
|   |                        Runtime Dispatcher                           |   |
|   +---------------------------------+-----------------------------------+   |
+-------------------------------------|---------------------------------------+
                                      |
                                      v
+-----------------------------------------------------------------------------+
|                          Core Engine Architecture                           |
|                                                                             |
|   +---------------------------------------------------------------------+   |
|   |                          Market Watcher                             |   |
|   |  - Active Contract Polling & WebSocket Event Ingestion              |   |
|   |  - Strike Price, Expiry Countdown & Order Book Depth Streaming     |   |
|   +---------------------------------+-----------------------------------+   |
|                                     |                                       |
|                                     v                                       |
|   +---------------------------------------------------------------------+   |
|   |                         Strategy Evaluator                          |   |
|   |  - Spread Arbitrage Engine (Bid-Ask Imbalance Calculation)          |   |
|   |  - Parity Edge Checker (UP Price + DOWN Price < 1.00 Parity)        |   |
|   |  - Risk Filters: Min Edge, Max Bet Sizing, Expiry Cutoff Buffer     |   |
|   +---------------------------------+-----------------------------------+   |
|                                     |                                       |
|                                     v                                       |
|   +---------------------------------------------------------------------+   |
|   |                        Transaction Manager                          |   |
|   |  - Atomic Local Nonce Sequencer                                     |   |
|   |  - Dynamic Gas & Gas-Ceiling Buffering                              |   |
|   |  - Secp256k1 Local Transaction Signer                               |   |
|   +---------------------------------+-----------------------------------+   |
|                                     |                                       |
|                                     v                                       |
|   +---------------------------------------------------------------------+   |
|   |                     Settlement & Sweeper Daemon                     |   |
|   |  - Resolution Oracle State Detection                                |   |
|   |  - Autonomous Payout Redemption Dispatcher                          |   |
|   +---------------------------------------------------------------------+   |
+-------------------------------------|---------------------------------------+
                                      |
                                      v
+-----------------------------------------------------------------------------+
|                         Protocol Driver Layer (Adapters)                    |
|                                                                             |
|   +-------------------------------------+   +---------------------------+   |
|   |      Somnia / DreamDEX Adapter      |   | Polymarket / EVM Adapter  |   |
|   |  - Chain ID: 50312 (Testnet)        |   | (Secondary Plug-in Target)|   |
|   |  - abigen Generated Contract Stubs  |   +---------------------------+   |
|   |  - On-chain CLOB & Settlement Calls |                                   |
|   +------------------+------------------+                                   |
+----------------------|------------------------------------------------------+
                       |
                       v Raw EVM JSON-RPC / WS
+-----------------------------------------------------------------------------+
|                          Target Blockchain Network                          |
|                     (Somnia Shannon EVM Testnet Node)                       |
+-----------------------------------------------------------------------------+
```

---

## 4. Repository & Package Layout

```
beaverish/
├── cmd/
│   └── agent/
│       └── main.go                 Entrypoint: parses flags, routes CLI/MCP/daemon modes
├── config/
│   ├── config.go                   Configuration schema and XDG-compliant file loader
│   └── defaults.go                 Default network endpoints, chain IDs, and risk gates
├── pkg/
│   ├── types/
│   │   ├── market.go               Domain definitions: MarketSnapshot, Order, Side
│   │   ├── adapter.go              Driver interfaces: MarketAdapter, OrderExecutor
│   │   └── strategy.go             Evaluation interface and Signal payload structures
│   ├── engine/
│   │   ├── engine.go               Core lifecycle coordinator (Start, Stop, Subscribe)
│   │   ├── watcher.go              Active market state poller and event listener
│   │   ├── executor.go             Trade dispatching pipeline and execution worker pool
│   │   ├── settler.go              Background resolution listener and payout sweeper
│   │   └── nonce.go                Thread-safe in-memory nonce manager
│   ├── strategy/
│   │   ├── spread_arb.go           Implementation: Spread and parity anomaly evaluation
│   │   └── momentum.go             Implementation: Fast-feed reference price divergence
│   ├── signer/
│   │   ├── signer.go               EVM private key transactor and signature wrapper
│   │   └── allowance.go            Automated ERC-20 token approval validator
│   ├── mcp/
│   │   ├── server.go               POSIX stdio JSON-RPC 2.0 transport loop
│   │   ├── protocol.go             MCP protocol message formats (tools/list, tools/call)
│   │   └── handlers.go             Tool call implementations for external agents
│   └── logger/
│       └── logger.go               High-signal formatted terminal and stderr stream logger
├── drivers/
│   └── somnia/
│       ├── bindings/
│       │   ├── dreamdex_clob.go    Generated Go bindings from DreamDEX CLOB ABI
│       │   ├── dreamdex_event.go   Generated Go bindings from Event Contract ABI
│       │   └── erc20.go            Generated standard ERC-20 interface bindings
│       ├── adapter.go              Somnia implementation of types.MarketAdapter
│       └── client.go               ethclient wrapper with retry and backoff semantics
├── docs/
│   ├── ARCHITECTURE.md             This system architecture blueprint
│   ├── MCP.md                      Detailed MCP protocol specification & integration examples
│   ├── SDK_FEEDBACK.md             Technical evaluation of protocol interfaces and SDKs
│   └── index.html                  Static GitHub Pages documentation portal
├── install.sh                      POSIX-compliant curl-to-shell installer script
├── Dockerfile                      Multi-stage minimal Alpine/scratch container definition
├── Makefile                        Build, test, abigen code generation, and lint recipes
├── README.md                       User and developer guide with one-command install & MCP setup
├── go.mod                          Go module definition
└── go.sum                          Checksums for Go module dependencies
```

---

## 5. Domain Models & Component Interfaces

### 5.1 Core Types & Domain Models

```go
package types

import (
    "context"
    "math/big"
)

type OutcomeSide uint8

const (
    OutcomeUp OutcomeSide = iota
    OutcomeDown
)

type MarketStatus uint8

const (
    MarketStatusActive MarketStatus = iota
    MarketStatusLocked
    MarketStatusResolved
)

type MarketSnapshot struct {
    MarketID        string       `json:"market_id"`
    ContractAddress string       `json:"contract_address"`
    UnderlyingAsset string       `json:"underlying_asset"`
    StrikePrice     *big.Int     `json:"strike_price"`
    ExpiryTimestamp int64        `json:"expiry_timestamp"`
    LockTimestamp   int64        `json:"lock_timestamp"`
    BestBidUp       *big.Int     `json:"best_bid_up"`
    BestAskUp       *big.Int     `json:"best_ask_up"`
    BestBidDown     *big.Int     `json:"best_bid_down"`
    BestAskDown     *big.Int     `json:"best_ask_down"`
    Status          MarketStatus `json:"status"`
    WinningOutcome  OutcomeSide  `json:"winning_outcome"`
}

type Signal struct {
    ShouldExecute bool        `json:"should_execute"`
    MarketID      string      `json:"market_id"`
    TargetSide    OutcomeSide `json:"target_side"`
    TargetPrice   *big.Int    `json:"target_price"`
    PositionSize  *big.Int    `json:"position_size"`
    Reason        string      `json:"reason"`
}

type AccountStatus struct {
    Address           string   `json:"address"`
    NativeBalance     *big.Int `json:"native_balance"`
    CollateralBalance *big.Int `json:"collateral_balance"`
    OpenPositions     int      `json:"open_positions"`
    RealizedPnL       *big.Int `json:"realized_pnl"`
}
```

### 5.2 Protocol Driver Interface

```go
type MarketAdapter interface {
    Name() string
    GetActiveMarkets(ctx context.Context) ([]MarketSnapshot, error)
    GetMarketSnapshot(ctx context.Context, marketID string) (*MarketSnapshot, error)
    PlaceLimitOrder(ctx context.Context, marketID string, side OutcomeSide, price *big.Int, amount *big.Int) (string, error)
    PlaceMarketOrder(ctx context.Context, marketID string, side OutcomeSide, amount *big.Int) (string, error)
    ClaimWinningPayout(ctx context.Context, marketID string) (string, error)
    EnsureAllowance(ctx context.Context, tokenAddress string, spenderAddress string, minAmount *big.Int) (string, error)
    GetAccountStatus(ctx context.Context) (*AccountStatus, error)
}
```

### 5.3 Strategy Interface

```go
type Strategy interface {
    Name() string
    Evaluate(snapshot MarketSnapshot) Signal
}
```

---

## 6. Execution Lifecycle & Subsystems

### 6.1 State Machine Lifecycle

1. Market Ingestion:
   - Poller queries active event contract registry every configured interval (e.g., 500ms).
   - Extracts current order book levels: Best Bid/Ask for UP and DOWN outcome tokens.

2. Risk & Timing Pre-Filter:
   - Evaluates remaining time to market lock:
     `TimeRemaining = ExpiryTimestamp - CurrentTimestamp`
   - Safety Threshold: If `TimeRemaining <= ExpiryCutoffSeconds` (e.g., 30s), reject order creation and transition to settlement monitoring.

3. Opportunity Assessment:
   - Spread Check: Validates that `Ask(UP) + Ask(DOWN) < 1.00 - EdgeThreshold`.
   - Sizing: Enforces `MaxBetSize` and `AvailableAccountCollateral` bounds.

4. Nonce Allocation & Signing:
   - Fetches atomic nonce from internal cache (synced with `PendingNonceAt` on boot).
   - Sets gas limit with a fixed 25% safety overhead.
   - Signs transaction locally via secp256k1 private key.

5. Broadcast & Receipt Tracking:
   - Broadcasts signed raw transaction to Somnia JSON-RPC.
   - Spawns asynchronous receipt verifier to track inclusion in target block.

6. Settlement & Payout Sweep:
   - Background daemon tracks all open positions against market expiry timers.
   - When `MarketStatus == MarketStatusResolved` and position outcome is winning, calls `ClaimWinningPayout()`.

### 6.2 Sequence Diagram

```
Operator/AI           Engine Runner          MarketAdapter (Somnia)        EVM Node
     |                      |                          |                      |
     |  Start Daemon        |                          |                      |
     |--------------------->|                          |                      |
     |                      | EnsureAllowance()        |                      |
     |                      |------------------------->| eth_sendRawTx        |
     |                      |                          |--------------------->|
     |                      |                          |<---------------------|
     |                      | Poll Active Markets      |                      |
     |                      |------------------------->| eth_call             |
     |                      |                          |--------------------->|
     |                      |                          |<---------------------|
     |                      | Evaluate Strategy        |                      |
     |                      | [Edge Detected]          |                      |
     |                      |                          |                      |
     |                      | PlaceLimitOrder()        |                      |
     |                      |------------------------->| eth_sendRawTransaction
     |                      |                          |--------------------->|
     |                      |                          | Tx Hash: 0x9f8c...   |
     |                      |                          |<---------------------|
     |                      | [Expiry Reached]         |                      |
     |                      | Check Resolution         |                      |
     |                      |------------------------->| eth_call             |
     |                      |                          |<---------------------|
     |                      | ClaimWinningPayout()     |                      |
     |                      |------------------------->| eth_sendRawTransaction
     |                      |                          |--------------------->|
     |                      | Log Settlement & PnL     |                      |
```

---

## 7. Model Context Protocol (MCP) Integration

When launched with the `--mcp` flag, the binary suspends direct stdout console logging and activates a JSON-RPC 2.0 server operating over POSIX standard streams:

- Input: Reads newline-delimited JSON-RPC messages from `os.Stdin`.
- Output: Writes newline-delimited JSON-RPC responses to `os.Stdout`.
- Diagnostics: All diagnostic, telemetry, and execution logs route strictly to `os.Stderr`.

### 7.1 Exposed MCP Tools

1. `get_markets`:
   - Description: Retrieves all active binary event contracts on Somnia with strike prices, expirations, and current bid/ask depths.
   - Parameters: `underlying` (string, optional: "BTC" | "ETH").

2. `evaluate_market`:
   - Description: Runs deterministic spread and probability evaluation on a target market to check for mispricing.
   - Parameters: `market_id` (string, required).

3. `execute_order`:
   - Description: Signs and submits an order to buy binary outcome shares (UP or DOWN) on Somnia testnet.
   - Parameters: `market_id` (string), `side` (string: "UP" | "DOWN"), `amount` (number), `price_limit` (number).

4. `sweep_settlements`:
   - Description: Inspects open positions, checks for resolved winning contracts, and executes payout claims.
   - Parameters: None.

5. `get_account_status`:
   - Description: Returns testnet gas token balance, collateral balance, open positions, and realized PnL.
   - Parameters: None.

---

## 8. Configuration & State Management

The engine respects the XDG Base Directory Specification, resolving configuration via a strict precedence ladder:
1. Explicit CLI Flags (`--key`, `--rpc`, `--max-bet`)
2. Environment Variables (`AGENT_PRIVATE_KEY`, `AGENT_RPC_URL`, etc.)
3. Configuration File (`$XDG_CONFIG_HOME/beaverish/config.json` or `~/.config/beaverish/config.json` or `./config.json`)
4. Static Compiled Defaults

### 8.1 Configuration Schema

```json
{
  "network": {
    "driver": "somnia",
    "rpc_url": "https://dream-rpc.somnia.network",
    "ws_url": "wss://dream-rpc.somnia.network/ws",
    "chain_id": 50312,
    "contracts": {
      "clob_router": "0x0000000000000000000000000000000000000000",
      "collateral_token": "0x0000000000000000000000000000000000000000"
    }
  },
  "wallet": {
    "private_key": "",
    "auto_approve": true
  },
  "risk": {
    "max_bet_size_units": 5.0,
    "max_drawdown_limit_units": 50.0,
    "min_edge_threshold": 0.04,
    "expiry_cutoff_seconds": 30
  },
  "runtime": {
    "poll_interval_ms": 500,
    "dry_run": false,
    "log_format": "text"
  }
}
```

---

## 9. Error Recovery & Safety Gates

1. Nonce Management:
   - Internal counter synchronized on boot.
   - If an RPC returns `NonceTooLow` or `ReplacementTransactionUnderpriced`, the engine initiates an immediate on-chain nonce re-synchronization.

2. Volatility & Expiry Cutoff:
   - Orders are strictly rejected if remaining market duration is less than `expiry_cutoff_seconds` to prevent trade execution during oracle resolution transitions.

3. Allowance Pre-flight:
   - Initialization cycle verifies ERC-20 collateral allowance against the CLOB router contract. If allowance < required, an `approve()` transaction is submitted and confirmed before trading starts.

4. Drawdown Circuit Breaker:
   - In-memory tracker tallies net session loss. If loss exceeds `max_drawdown_limit_units`, the daemon halts execution, triggers an alarm log, and shifts strictly to sweeping open winning settlements.

---

## 10. Build, Deployment & Distribution

### 10.1 Compilation
Targeting a single, zero-dependency static binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/beaverish ./cmd/agent
```

### 10.2 Single-Command Shell Installation
A POSIX install script downloads the precompiled platform binary or builds from source, places it in `~/.local/bin/beaverish`, sets execution permissions, and scaffolds the default config template in `~/.config/beaverish/config.json`.

### 10.3 Containerization
Multi-stage build utilizing `golang:alpine` as builder and a minimal, non-root `alpine` runtime container with GHCR publishing workflow.
