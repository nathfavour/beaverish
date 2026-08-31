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
	"github.com/nathfavour/beaverish/pkg/engine"
	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/mcp"
	"github.com/nathfavour/beaverish/pkg/signer"
	"github.com/nathfavour/beaverish/pkg/tui"
)

var (
	version = "1.0.0"
)

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

	// In MCP or TUI mode, stdout is reserved; logs go to stderr
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
		logger.Infof("Running without private key (Read-Only / Simulation mode)")
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

	// Autonomous daemon execution
	_ = daemonMode
	if err := coreEngine.Start(ctx); err != nil {
		logger.Errorf("Engine start error: %v", err)
		os.Exit(1)
	}

	logger.Infof("Beaverish daemon active. Watching event markets... Press Ctrl+C to terminate.")
	<-ctx.Done()
	coreEngine.Stop()
}
