---
name: dogfooding
description: >-
  Dogfood and verify Beaverish locally using CLI simulation, mock feeds, and direct MCP stdio JSON-RPC sessions.
---

# Beaverish Dogfooding & Verification Guide

Follow these practices to dogfood Beaverish directly within this workspace.

---

## 1. Test MCP Stdio JSON-RPC Locally

Verify the MCP server handshake, tool listing, and market query in a single piped pipeline:

```bash
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_markets","arguments":{}}}\n' | ./bin/beaverish --mcp
```

## 2. Test Market Evaluation & Order Execution Simulation

```bash
# Dry-run daemon execution with verbose debug output
./bin/beaverish --dry-run --debug
```

## 3. Unit Test Verification with Race Detection

```bash
go test -v -race ./...
```
