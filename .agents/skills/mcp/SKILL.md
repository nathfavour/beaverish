---
name: mcp
description: >-
  Connect AI agents, Claude Code, Cursor, Codex, and Antigravity to the Beaverish
  Model Context Protocol (MCP) server for binary event contract trading and settlement.
---

# Beaverish Model Context Protocol (MCP) Server

Connect AI coding agents, autonomous runtimes, and IDEs to Beaverish via POSIX stdio JSON-RPC 2.0.

---

## 1. Quick Start

### Installation

```bash
# Direct single-command install from source
curl -sSL https://raw.githubusercontent.com/nathfavour/beaverish/main/install.sh | sh
```

### IDE / Agent Config

#### Claude Desktop / Claude Code (`claude_desktop_config.json`)
```json
{
  "mcpServers": {
    "beaverish": {
      "command": "beaverish",
      "args": ["--mcp"],
      "env": {
        "AGENT_PRIVATE_KEY": "0xYOUR_HEX_PRIVATE_KEY",
        "AGENT_RPC_URL": "https://dream-rpc.somnia.network",
        "AGENT_DRY_RUN": "false"
      }
    }
  }
}
```

#### Antigravity (`~/.gemini/antigravity-cli/mcp_config.json`)
```json
{
  "mcpServers": {
    "beaverish": {
      "command": "beaverish",
      "args": ["--mcp"]
    }
  }
}
```

---

## 2. Available MCP Tools

| Tool | Parameters | Description |
|---|---|---|
| `get_markets` | `underlying` *(string, optional)* | Retrieve active event contracts & orderbook depths on Somnia. |
| `evaluate_market` | `market_id` *(string, required)* | Run spread & parity edge ($Ask_{UP} + Ask_{DOWN} < 1.00$) evaluation. |
| `execute_order` | `market_id`, `side`, `amount`, `price_limit` | Dispatch limit or market orders. |
| `sweep_settlements`| *None* | Claim winning contract payouts. |
| `get_account_status`| *None* | Query gas balance, collateral balance, and open positions. |

---

## 3. Stdio Protocol Spec

- **Transport**: Standard I/O (JSON-RPC 2.0 over `stdin`/`stdout`).
- **Logs**: All logs route to `stderr`.
