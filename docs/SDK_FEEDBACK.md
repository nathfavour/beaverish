# Protocol & SDK Feedback: Somnia / DreamDEX Integration

## 1. Executive Summary
This document provides architectural evaluation and technical feedback for integrating high-frequency binary event contracts and order-book prediction markets on the Somnia EVM network (Chain ID: 50312).

## 2. Key Findings & Optimizations

### 2.1 Sub-second Block Times & Atomic Nonce Alignment
- Somnia's high throughput requires deterministic local nonce management. Relying on remote `eth_getTransactionCount` on every trade introduces pipeline stalls. Beaverish implements an atomic in-memory nonce sequencer synchronized on boot and resynchronized only on transaction underprice or drop events.

### 2.2 Event Contract CLOB Interaction
- The DreamDEX CLOB architecture separates limit order placing, matching, and settlement redemption.
- Pre-flight automated ERC-20 allowances prevent runtime trade rejection.
- Parity edge detection ($Ask_{UP} + Ask_{DOWN} < 1.00$) reliably surfaces mispriced binary outcome tokens.

### 2.3 POSIX stdio MCP Protocol Standardization
- Separating stdio JSON-RPC 2.0 messages from operational logs (redirected to `stderr`) ensures full compatibility with Claude Code, Cursor, Antigravity, and Codex agent runtimes.
