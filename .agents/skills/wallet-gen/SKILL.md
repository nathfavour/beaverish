---
name: wallet-gen
description: >-
  Generate and derive EVM wallets, private keys, and mnemonic seeds for Beaverish testing
  and production execution using hdwallet-cli.
---

# EVM Wallet Generation & HD Derivation for Beaverish

This skill provides step-by-step guidance on generating, inspecting, and managing EVM-compatible wallets and private keys for the Beaverish daemon and MCP server on the Somnia network (Chain ID: 50312).

---

## 1. Tooling: `hdwallet-cli`

We use `hdwallet-cli` for instant, offline, deterministic key generation.

### Installation on Fresh Testing Environments
If `hdwallet-cli` is not yet installed on your machine, install it via Go:

```bash
go install github.com/X-Vlad/hdwallet-cli@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `$PATH`:
```bash
export PATH="$PATH:$HOME/go/bin"
```

---

## 2. Wallet Generation Recipes

### A. Generate a New Mnemonic & First EVM Key
```bash
hdwallet-cli gen -n 1
```

### B. Extract Private Key & Public Address
```bash
# Derive specific standard EVM path: m/44'/60'/0'/0/0
hdwallet-cli gen -n 1 --path "m/44'/60'/0'/0/0"
```

### C. Direct Integration with Beaverish
Export the generated private key for the daemon or MCP runtime:

```bash
export AGENT_PRIVATE_KEY="0xYOUR_HEX_PRIVATE_KEY"
export AGENT_RPC_URL="https://dream-rpc.somnia.network"
```

Or write into Beaverish configuration:
```json
{
  "wallet": {
    "private_key": "0xYOUR_HEX_PRIVATE_KEY",
    "auto_approve": true
  }
}
```

---

## 3. Safety Guardrails
- NEVER commit real private keys or mnemonic phrases into Git repositories.
- Use environment variables (`AGENT_PRIVATE_KEY` or `BEAVERISH_PRIVATE_KEY`) or XDG config files (`~/.config/beaverish/config.json`).
