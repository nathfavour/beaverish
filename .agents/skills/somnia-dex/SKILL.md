---
name: somnia-dex
description: >-
  Somnia Shannon EVM testnet (Chain ID 50312) DreamDEX CLOB and Event Contract driver guide.
---

# Somnia DreamDEX & Event Contract Integration

## 1. Network Parameters
- **Chain ID**: `50312`
- **RPC URL**: `https://dream-rpc.somnia.network`
- **WebSocket URL**: `wss://dream-rpc.somnia.network/ws`

## 2. Parity Arbitrage Edge
Binary event markets price outcomes on a $[0, 1.00]$ unit scale ($1e18$ wei).
When:
$$Ask_{UP} + Ask_{DOWN} < 1.00 - \text{EdgeThreshold}$$
an instantaneous risk-neutral parity edge exists. Beaverish detects this via `SpreadArbStrategy` and routes fill orders.

## 3. Allowance & Settlement Flow
1. **Pre-flight**: `EnsureAllowance()` ensures CLOB router can pull collateral tokens without in-flight transaction halts.
2. **Order Execution**: Orders are dispatched via atomic local nonce sequence.
3. **Settlement**: When lock or expiry is reached and resolution is reported by oracle, `ClaimWinningPayout()` redeems the 1.00 payout token.
