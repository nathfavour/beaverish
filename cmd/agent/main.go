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
	version = "1.0.1"
)

func printBanner(cfg *config.Config, walletAddr string) {
	cyan := "\033[36m"
	bold := "\033[1m"
	green := "\033[32m"
	reset := "\033[0m"
	dim := "\033[90m"

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  %s%s╔════════════════════════════════════════════════════════════════════════╗%s\n", bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s%s║               🦫 BEAVERISH HIGH-THROUGHPUT EVENT ENGINE               ║%s\n", bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s%s║       Arbitrage, CLOB Settlement & MCP Stdio Streaming Daemon          ║%s\n", bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s%s╚════════════════════════════════════════════════════════════════════════╝%s\n", bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "  %sVersion:%s v%-6s  %sNetwork:%s %-12s  %sChain ID:%s %-6d\n", dim, reset, version, dim, reset, cfg.Network.Driver, dim, reset, cfg.Network.ChainID)
	fmt.Fprintf(os.Stderr, "  %sRPC:%s %-32s  %sMax Bet:%s %.1f Units\n", dim, reset, cfg.Network.RPCURL, dim, reset, cfg.Risk.MaxBetSizeUnits)
	if walletAddr != "" {
		fmt.Fprintf(os.Stderr, "  %sTransactor:%s %s%s%s\n", dim, reset, green, walletAddr, reset)
	}
	fmt.Fprintf(os.Stderr, "  %s────────────────────────────────────────────────────────────────────────%s\n\n", dim, reset)
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
	walletGen := flag.Bool("wallet-gen", false, "Generate a new EVM wallet using hdwallet-cli or internal fallback")
	flag.Parse()

	if *verFlag {
		fmt.Printf("beaverish v%s\n", version)
		os.Exit(0)
	}

	if *walletGen || (len(os.Args) > 1 && os.Args[1] == "wallet") {
		w, err := signer.GenerateWallet()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating wallet: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated EVM Wallet (Source: %s)\n", w.Source)
		fmt.Printf("Address:     %s\n", w.Address)
		if w.Mnemonic != "" {
			fmt.Printf("Mnemonic:    %s\n", w.Mnemonic)
		}
		fmt.Printf("Private Key: %s\n", w.PrivateKey)
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

	// If another instance is running and headless daemon mode is requested, attach follower console
	if !isServer && *daemonMode && !*mcpMode {
		printBanner(cfg, "")
		logger.Section("FOLLOWER CONSOLE ATTACHED")
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

	var userSigner *signer.Signer
	var walletAddress string
	if cfg.Wallet.PrivateKey != "" {
		s, err := signer.NewSigner(cfg.Wallet.PrivateKey, cfg.Network.ChainID)
		if err != nil {
			logger.Warnf("Could not initialize signer: %v", err)
		} else {
			userSigner = s
			walletAddress = s.Address().Hex()
		}
	}

	runTUI := *tuiMode || (!*daemonMode && !*mcpMode)
	if !runTUI && !*mcpMode {
		printBanner(cfg, walletAddress)
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
		logger.Section("MCP SERVER ACTIVE")
		logger.Infof("Listening for tools/list, tools/call over POSIX stdio JSON-RPC 2.0...")
		handler := mcp.NewHandler(cfg, coreEngine)
		server := mcp.NewServer(os.Stdin, os.Stdout, handler)
		if err := server.Start(ctx); err != nil {
			logger.Errorf("MCP Server error: %v", err)
			os.Exit(1)
		}
		return
	}

	// Interactive TUI runs when -tui is specified, OR by default when run interactively without -daemon
	if runTUI {
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

	// Headless Daemon execution loop (-daemon flag)
	if err := coreEngine.Start(ctx); err != nil {
		logger.Errorf("Engine start error: %v", err)
		os.Exit(1)
	}

	logger.Section("ACTIVE MONITORING LOOP")
	logger.Infof("Streaming active markets, checking parity arbitrage & sweep payouts...")
	logger.Infof("Hot-reload directory: %s", daemon.ConfigDir())
	logger.Infof("Press Ctrl+C to terminate cleanly.\n")
	<-ctx.Done()
	coreEngine.Stop()
}
