package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/drivers/somnia"
	"github.com/nathfavour/beaverish/pkg/daemon"
	"github.com/nathfavour/beaverish/pkg/engine"
	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/mcp"
	"github.com/nathfavour/beaverish/pkg/signer"
	"github.com/nathfavour/beaverish/pkg/tui"
)

var (
	version = "1.0.0"
)

func printBanner() {
	banner := `
  ╔═══════════════════════════════════════════════════════════╗
  ║    🦫 BEAVERISH — Somnia EVM Event Trading & MCP Daemon   ║
  ║    Version: v1.0.0 | Chain ID: 50312 (Somnia Testnet)     ║
  ║    Mode: Autonomous Arbitrage, Sweeper & Real-Time Engine ║
  ╚═══════════════════════════════════════════════════════════╝
`
	fmt.Fprintln(os.Stderr, banner)
}

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	mcpMode := flag.Bool("mcp", false, "Run as Model Context Protocol (MCP) server over POSIX stdio")
	tuiMode := flag.Bool("tui", false, "Run with interactive Lipgloss / Bubbletea TUI dashboard")
	daemonMode := flag.Bool("daemon", false, "Run in headless autonomous trading loop")
	dryRun := flag.Bool("dry-run", false, "Enable dry-run mode (no real transactions submitted)")
	privateKey := flag.String("key", "", "Hex private key for EVM transactor")
	rpcURL := flag.String("rpc", "", "RPC endpoint for target chain")
	maxBet := flag.Float64("max-bet", 0, "Max bet size in standard units")
	debug := flag.Bool("debug", false, "Enable verbose debug logs")
	verFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *verFlag {
		fmt.Printf("beaverish v%s\n", version)
		os.Exit(0)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if *privateKey != "" {
		cfg.Wallet.PrivateKey = *privateKey
	}
	if *rpcURL != "" {
		cfg.Network.RPCURL = *rpcURL
	}
	if *maxBet > 0 {
		cfg.Risk.MaxBetSizeUnits = *maxBet
	}
	if *dryRun {
		cfg.Runtime.DryRun = true
	}
	if *debug {
		cfg.Runtime.Debug = true
	}

	// In MCP or TUI mode, stdout is strictly reserved; logs route to stderr
	logger.Init(os.Stderr, cfg.Runtime.LogFormat, cfg.Runtime.Debug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Infof("Shutdown signal received. Terminating safely...")
		cancel()
	}()

	// Watch config directory and update file for real-time live reload
	daemon.WatchForUpdates(ctx, func() {
		logger.Infof("Live update triggered! Preparing in-place reload...")
	})

	// Single-Instance Arbitration & Centralized Closed Feedback Loop
	singleInst, isServer, err := daemon.AcquireOrConnect("")
	if err != nil {
		logger.Warnf("Single instance coordinator warning: %v", err)
	}

	// If running CLI in secondary terminal and NOT in MCP mode, attach as live stream follower
	if !isServer && !*mcpMode && !*tuiMode {
		printBanner()
		logger.Infof("Primary Beaverish engine is active! Attaching follower console to live stream...")
		if err := daemon.AttachFollower(ctx, ""); err != nil {
			logger.Errorf("Follower stream closed: %v", err)
		}
		return
	}

	if singleInst != nil && singleInst.IsServer() {
		defer singleInst.Close()
		singleInst.StartBroadcaster(ctx)
		// Hook logger so every event/telemetry/MCP log is broadcasted to attached follower shells
		logger.AddHook(func(level logger.Level, prefix, message string) {
			singleInst.Broadcast("log", fmt.Sprintf("[%s] %s", prefix, message))
		})
	}

	if !*tuiMode {
		printBanner()
	}

	var userSigner *signer.Signer
	if cfg.Wallet.PrivateKey != "" {
		s, err := signer.NewSigner(cfg.Wallet.PrivateKey, cfg.Network.ChainID)
		if err != nil {
			logger.Warnf("Could not initialize signer: %v", err)
		} else {
			userSigner = s
			logger.Infof("Transactor Signer initialized. Address: %s", s.Address().Hex())
		}
	} else {
		logger.Infof("Operating in Read-Only / Simulation mode (Set AGENT_PRIVATE_KEY for live transactions)")
	}

	var nonceMgr *engine.NonceManager
	if userSigner != nil {
		nonceMgr = engine.NewNonceManager(nil, userSigner.Address())
	}

	adapter, err := somnia.NewSomniaAdapter(cfg, userSigner, nonceMgr)
	if err != nil {
		logger.Errorf("Failed to initialize Somnia adapter: %v", err)
		os.Exit(1)
	}

	coreEngine := engine.NewEngine(cfg, adapter)

	if *mcpMode {
		logger.Infof("Starting Beaverish MCP Server over POSIX stdio (JSON-RPC 2.0)...")
		handler := mcp.NewHandler(cfg, coreEngine)
		server := mcp.NewServer(os.Stdin, os.Stdout, handler)
		if err := server.Start(ctx); err != nil {
			logger.Errorf("MCP Server error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *tuiMode {
		if err := coreEngine.Start(ctx); err != nil {
			logger.Errorf("Engine start error: %v", err)
			os.Exit(1)
		}
		if err := tui.RunTUI(cfg, coreEngine); err != nil {
			logger.Errorf("TUI error: %v", err)
			os.Exit(1)
		}
		coreEngine.Stop()
		return
	}

	// Default CLI daemon execution: starts primary engine loop
	_ = daemonMode
	if err := coreEngine.Start(ctx); err != nil {
		logger.Errorf("Engine start error: %v", err)
		os.Exit(1)
	}

	logger.Infof("Beaverish daemon active. Watching event markets & arbitrage opportunities...")
	logger.Infof("Network: %s | RPC: %s | Max Bet: %.1f Units", cfg.Network.Driver, cfg.Network.RPCURL, cfg.Risk.MaxBetSizeUnits)
	logger.Infof("Watching config & update triggers at: %s", daemon.ConfigDir())
	logger.Infof("Press Ctrl+C to terminate cleanly.")
	<-ctx.Done()
	coreEngine.Stop()
}
