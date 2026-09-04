# Beaverish Model Context Protocol (MCP) Integration

Beaverish provides a native Model Context Protocol (MCP) server running over POSIX standard streams (`stdio`), adhering to the JSON-RPC 2.0 specification.

This enables direct, zero-friction integration into agentic workflows and AI coding runtimes such as:
- **Claude Code / Desktop**
- **Codex CLI / Assistants**
- **Antigravity CLI / IDE**
- **Cursor IDE**
- **Custom Autonomous Orchestrators**

---

## 1. Quick MCP Configuration

### Claude Desktop / Claude Code (`claude_desktop_config.json`)

Add the following to your Claude configuration file (e.g. `~/.config/Claude/claude_desktop_config.json` or `%APPDATA%/Claude/claude_desktop_config.json`):

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

Or using Docker / GHCR:

```json
{
  "mcpServers": {
    "beaverish": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "AGENT_PRIVATE_KEY=0xYOUR_HEX_PRIVATE_KEY",
        "-e", "AGENT_RPC_URL=https://dream-rpc.somnia.network",
        "ghcr.io/nathfavour/beaverish:latest",
        "--mcp"
      ]
    }
  }
}
```

### Antigravity (`~/.gemini/antigravity-cli/mcp_config.json` or workspace settings)

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

### `get_markets`
Retrieves all active binary event contracts on Somnia with strike prices, expirations, and current bid/ask depths.

- **Parameters**:
  - `underlying` *(string, optional)*: Filter by underlying asset symbol, e.g. `"BTC"` or `"ETH"`.

### `evaluate_market`
Runs deterministic spread and probability evaluation on a target market to check for mispricing and parity arbitrage edge ($Ask(UP) + Ask(DOWN) < 1.00$).

- **Parameters**:
  - `market_id` *(string, required)*: Hex ID of the binary event market.

### `execute_order`
Signs and broadcasts a limit or market order to buy binary outcome shares (UP or DOWN) on Somnia testnet.

- **Parameters**:
  - `market_id` *(string, required)*: ID of the market to trade.
  - `side` *(string, required)*: `"UP"` or `"DOWN"`.
  - `amount` *(number, required)*: Size in units (e.g. `1.0`, `5.0`).
  - `price_limit` *(number, optional)*: Limit price (e.g. `0.48`). If omitted or 0, executes as a market order.

### `cancel_order`
Cancels an open limit order on Somnia DreamDEX CLOB.

- **Parameters**:
  - `order_id` *(string, required)*: Hex ID of the open order to cancel.

### `get_orderbook`
Retrieves full bid and ask depth (prices and amounts for both UP and DOWN outcome tokens) for a market.

- **Parameters**:
  - `market_id` *(string, required)*: ID of the market to inspect.

### `get_positions`
Lists all current open and settled trade positions with entry prices, amounts, and settlement statuses.

- **Parameters**: None.

### `sweep_settlements`
Inspects open positions, checks for resolved winning contracts, and executes payout claims.

- **Parameters**: None.

### `get_account_status`
Returns testnet gas token balance, collateral balance, open positions count, and realized PnL.

- **Parameters**: None.

### `get_config`
Returns active runtime configuration parameters, target network settings, and risk limits.

- **Parameters**: None.

---

## 3. Protocol Message Flow Example

### Initialize Server
```json
--> {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
<-- {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":true}},"protocolVersion":"2024-11-05","serverInfo":{"name":"beaverish-mcp","version":"1.0.0"}}}
```

### List Tools
```json
--> {"jsonrpc":"2.0","id":2,"method":"tools/list"}
<-- {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"get_markets",...},{"name":"execute_order",...}]}}
```

### Call Tool: `get_markets`
```json
--> {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_markets","arguments":{"underlying":"BTC"}}}
<-- {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"[{\"market_id\":\"0x...\",\"underlying_asset\":\"BTC\",...}]"}]}}
```
