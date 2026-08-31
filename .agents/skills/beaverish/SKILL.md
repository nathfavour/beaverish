---
name: beaverish
description: >-
  Core Beaverish autonomous daemon, architecture, signal evaluators, and lifecycle coordinator.
---

# Beaverish Engine Core Architecture

Refer to [ARCHITECTURE.md](../../ARCHITECTURE.md) for full topology.

## Core Packages
- `pkg/types`: Domain snapshots, signals, and interface definitions.
- `pkg/engine`: `MarketWatcher`, `ExecutionPipeline`, `SettlementDaemon`, and `NonceManager`.
- `pkg/strategy`: Spread arbitrage and momentum evaluators.
- `pkg/signer`: Secp256k1 EVM transactor and token allowance manager.
- `pkg/mcp`: POSIX stdio JSON-RPC server.
- `drivers/somnia`: Protocol adapter for Somnia network.
