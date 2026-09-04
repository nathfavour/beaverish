# 🦫 Beaverish

[![Build and Publish GHCR Container](https://github.com/nathfavour/beaverish/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/nathfavour/beaverish/actions/workflows/docker-publish.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Beaverish** is a high-throughput, protocol-agnostic daemon written in Go for executing, managing, and settling high-frequency binary event contracts and order-book prediction markets.

While designed with a modular driver abstraction that decouples execution logic from specific blockchain platforms, the primary reference implementation targets **DreamDEX Event Contracts** deployed on the high-throughput **Somnia EVM network (Chain ID: 50312)**.

The system operates in dual execution modes:
1. **Headless Autonomous Daemon**: A background engine evaluating deterministic statistical arbitrage, order-book parity ($Ask_{UP} + Ask_{DOWN} < 1.00$), and latency signals to automatically place, track, and sweep-settle binary outcome positions.
2. **Embedded Model Context Protocol (MCP) Server**: A standard POSIX stdio JSON-RPC 2.0 interface allowing external AI runtimes and automated orchestrators (Claude Code, Antigravity, Codex, Cursor) to inspect markets, assess risk, and dispatch signed transactions.

---

## ⚡ Installation

### Method 1: Single-Command Install (Anyisland + Beaverish)

Install both Anyisland and Beaverish in a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/nathfavour/anyisland/master/install.sh | bash -s -- nathfavour/beaverish
```

*(Already have Anyisland? Just run `anyisland install github.com/nathfavour/beaverish`)*

### Method 2: GitHub Container Registry (GHCR)

Run containerized without local Go installation:

```bash
docker run -i --rm ghcr.io/nathfavour/beaverish:latest --mcp
```

Or pull the prebuilt image:

```bash
docker pull ghcr.io/nathfavour/beaverish:latest
```

---

## 🤖 Model Context Protocol (MCP) Integration

Beaverish is built for **first-class integration** into AI orchestrators and coding tools like **Claude Code**, **Codex**, **Antigravity CLI / IDE**, and **Cursor**.

### Configuration

#### Claude Code / Claude Desktop (`claude_desktop_config.json`)
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

#### Antigravity CLI / IDE (`~/.gemini/antigravity-cli/mcp_config.json`)
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

### Exposed MCP Tools

| Tool Name | Parameters | Description |
| :--- | :--- | :--- |
| `get_markets` | `underlying` *(optional string: "BTC" \| "ETH")* | Retrieves all active event contracts on Somnia with strike prices, expirations, and bid/ask depths. |
| `evaluate_market` | `market_id` *(required string)* | Runs deterministic spread and parity edge evaluation ($Ask_{UP} + Ask_{DOWN} < 1.00$) on target market. |
| `execute_order` | `market_id`, `side` ("UP" \| "DOWN"), `amount`, `price_limit` | Signs and broadcasts a transaction order on Somnia testnet. |
| `sweep_settlements` | *None* | Inspects positions, checks for resolved winning contracts, and claims payouts. |
| `get_account_status`| *None* | Returns native testnet gas balance, collateral balance, open positions count, and realized PnL. |
| `get_config`        | *None* | Returns active runtime configuration, network settings, and risk limits. |

See [`docs/MCP.md`](docs/MCP.md) for full protocol specifications and request/response payloads.

---

## 🚀 Autonomous Daemon Usage

Run as a background autonomous trading daemon:

```bash
beaverish --key 0xYOUR_PRIVATE_KEY --rpc https://dream-rpc.somnia.network --max-bet 5.0
```

### Dry-Run / Read-Only Simulation
```bash
beaverish --dry-run --debug
```

### Available CLI Flags
- `--mcp`: Run in standard POSIX stdio JSON-RPC 2.0 MCP server mode (logs routed to stderr).
- `--daemon`: Run in headless autonomous loop (default).
- `--dry-run`: Simulation mode (no real on-chain transactions sent).
- `--key <hex>`: Hex EVM private key for transaction signing.
- `--rpc <url>`: Target EVM JSON-RPC endpoint (default: `https://dream-rpc.somnia.network`).
- `--max-bet <float>`: Maximum bet size in units (default: `5.0`).
- `--config <path>`: Custom path to `config.json`.
- `--debug`: Enable verbose debug logging.
- `--version`: Display version information.

---

## ⚙️ Configuration Schema

Beaverish adheres to the **XDG Base Directory Specification**, reading config from `$XDG_CONFIG_HOME/beaverish/config.json` (or `~/.config/beaverish/config.json`):

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

## 📚 Documentation & GitHub Pages

- [Architecture Blueprint](ARCHITECTURE.md) ([docs/ARCHITECTURE.md](docs/ARCHITECTURE.md))
- [MCP Integration Guide](docs/MCP.md)
- [Protocol & SDK Technical Feedback](docs/SDK_FEEDBACK.md)
- [Live Documentation Portal](docs/index.html)

---

## 🛠️ Development & Building

```bash
# Build binary to bin/beaverish
make build

# Run unit test suite
make test

# Build Docker image
make docker
```

---

## 📄 License
MIT License.
