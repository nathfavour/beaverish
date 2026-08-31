package config

const (
	DefaultDriver       = "somnia"
	DefaultRPCURL       = "https://dream-rpc.somnia.network"
	DefaultWSURL        = "wss://dream-rpc.somnia.network/ws"
	DefaultChainID      = int64(50312)
	DefaultCLOBRouter   = "0x0000000000000000000000000000000000000000"
	DefaultCollateral   = "0x0000000000000000000000000000000000000000"

	DefaultMaxBetSizeUnits      = 5.0
	DefaultMaxDrawdownUnits     = 50.0
	DefaultMinEdgeThreshold     = 0.04
	DefaultExpiryCutoffSec      = 30
	DefaultPollIntervalMs       = 500
	DefaultDryRun               = false
	DefaultLogFormat            = "text"
	DefaultAutoApprove          = true
)
