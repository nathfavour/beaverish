package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type NetworkConfig struct {
	Driver    string            `json:"driver"`
	RPCURL    string            `json:"rpc_url"`
	WSURL     string            `json:"ws_url"`
	ChainID   int64             `json:"chain_id"`
	Contracts map[string]string `json:"contracts"`
}

type WalletConfig struct {
	Address     string `json:"address,omitempty"`
	Mnemonic    string `json:"mnemonic,omitempty"`
	PrivateKey  string `json:"private_key"`
	AutoApprove bool   `json:"auto_approve"`
}

type RiskConfig struct {
	MaxBetSizeUnits      float64 `json:"max_bet_size_units"`
	MaxDrawdownLimitUnits float64 `json:"max_drawdown_limit_units"`
	MinEdgeThreshold     float64 `json:"min_edge_threshold"`
	ExpiryCutoffSeconds  int     `json:"expiry_cutoff_seconds"`
}

type RuntimeConfig struct {
	PollIntervalMs int    `json:"poll_interval_ms"`
	DryRun         bool   `json:"dry_run"`
	LogFormat      string `json:"log_format"`
	Debug          bool   `json:"debug"`
}

type Config struct {
	Network NetworkConfig `json:"network"`
	Wallet  WalletConfig  `json:"wallet"`
	Risk    RiskConfig    `json:"risk"`
	Runtime RuntimeConfig `json:"runtime"`
}

func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			Driver:  DefaultDriver,
			RPCURL:  DefaultRPCURL,
			WSURL:   DefaultWSURL,
			ChainID: DefaultChainID,
			Contracts: map[string]string{
				"clob_router":      DefaultCLOBRouter,
				"collateral_token": DefaultCollateral,
			},
		},
		Wallet: WalletConfig{
			PrivateKey:  "",
			AutoApprove: DefaultAutoApprove,
		},
		Risk: RiskConfig{
			MaxBetSizeUnits:       DefaultMaxBetSizeUnits,
			MaxDrawdownLimitUnits: DefaultMaxDrawdownUnits,
			MinEdgeThreshold:      DefaultMinEdgeThreshold,
			ExpiryCutoffSeconds:   DefaultExpiryCutoffSec,
		},
		Runtime: RuntimeConfig{
			PollIntervalMs: DefaultPollIntervalMs,
			DryRun:         DefaultDryRun,
			LogFormat:      DefaultLogFormat,
			Debug:          false,
		},
	}
}

func GetConfigPath() string {
	if custom := os.Getenv("BEAVERISH_CONFIG"); custom != "" {
		return custom
	}
	if custom := os.Getenv("AGENT_CONFIG"); custom != "" {
		return custom
	}
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			xdgConfig = filepath.Join(home, ".config")
		}
	}
	if xdgConfig != "" {
		p := filepath.Join(xdgConfig, "beaverish", "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		// Also check event-agent for backwards compatibility
		pOld := filepath.Join(xdgConfig, "event-agent", "config.json")
		if _, err := os.Stat(pOld); err == nil {
			return pOld
		}
		return p
	}
	return "config.json"
}

func LoadConfig(customPath string) (*Config, error) {
	cfg := DefaultConfig()

	path := customPath
	if path == "" {
		path = GetConfigPath()
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
	}

	// Environment variable overrides
	if pk := os.Getenv("AGENT_PRIVATE_KEY"); pk != "" {
		cfg.Wallet.PrivateKey = pk
	} else if pk := os.Getenv("BEAVERISH_PRIVATE_KEY"); pk != "" {
		cfg.Wallet.PrivateKey = pk
	}

	if mn := os.Getenv("AGENT_MNEMONIC"); mn != "" {
		cfg.Wallet.Mnemonic = mn
	} else if mn := os.Getenv("BEAVERISH_MNEMONIC"); mn != "" {
		cfg.Wallet.Mnemonic = mn
	}

	if addr := os.Getenv("AGENT_WALLET_ADDRESS"); addr != "" {
		cfg.Wallet.Address = addr
	} else if addr := os.Getenv("BEAVERISH_WALLET_ADDRESS"); addr != "" {
		cfg.Wallet.Address = addr
	}

	if rpc := os.Getenv("AGENT_RPC_URL"); rpc != "" {
		cfg.Network.RPCURL = rpc
	} else if rpc := os.Getenv("BEAVERISH_RPC_URL"); rpc != "" {
		cfg.Network.RPCURL = rpc
	}

	if ws := os.Getenv("AGENT_WS_URL"); ws != "" {
		cfg.Network.WSURL = ws
	} else if ws := os.Getenv("BEAVERISH_WS_URL"); ws != "" {
		cfg.Network.WSURL = ws
	}

	if chainID := os.Getenv("AGENT_CHAIN_ID"); chainID != "" {
		if id, err := strconv.ParseInt(chainID, 10, 64); err == nil {
			cfg.Network.ChainID = id
		}
	}

	if clob := os.Getenv("CLOB_ROUTER_ADDRESS"); clob != "" {
		cfg.Network.Contracts["clob_router"] = clob
	}
	if col := os.Getenv("COLLATERAL_TOKEN_ADDRESS"); col != "" {
		cfg.Network.Contracts["collateral_token"] = col
	}

	if maxBet := os.Getenv("AGENT_MAX_BET"); maxBet != "" {
		if v, err := strconv.ParseFloat(maxBet, 64); err == nil {
			cfg.Risk.MaxBetSizeUnits = v
		}
	}

	if dryRun := os.Getenv("AGENT_DRY_RUN"); dryRun != "" {
		cfg.Runtime.DryRun = dryRun == "true" || dryRun == "1"
	}

	return cfg, nil
}
