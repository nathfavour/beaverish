package engine

import (
	"context"
	"math/big"
	"time"

	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/strategy"
	"github.com/nathfavour/beaverish/pkg/types"
)

type Engine struct {
	cfg         *config.Config
	adapter     types.MarketAdapter
	watcher     *MarketWatcher
	executor    *ExecutionPipeline
	settler     *SettlementDaemon
	strategies  []types.Strategy
	marketCh    chan types.MarketSnapshot
	cancelFunc  context.CancelFunc
}

func NewEngine(cfg *config.Config, adapter types.MarketAdapter) *Engine {
	watcher := NewMarketWatcher(adapter, time.Duration(cfg.Runtime.PollIntervalMs)*time.Millisecond)
	executor := NewExecutionPipeline(cfg, adapter, 100)
	settler := NewSettlementDaemon(adapter, 5*time.Second)

	strats := []types.Strategy{
		strategy.NewSpreadArbStrategy(cfg.Risk.MinEdgeThreshold, cfg.Risk.MaxBetSizeUnits),
		strategy.NewMomentumStrategy(cfg.Risk.MaxBetSizeUnits),
	}

	return &Engine{
		cfg:        cfg,
		adapter:    adapter,
		watcher:    watcher,
		executor:   executor,
		settler:    settler,
		strategies: strats,
		marketCh:   make(chan types.MarketSnapshot, 100),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	e.cancelFunc = cancel

	logger.Infof("Starting Beaverish Core Engine [Driver: %s, ChainID: %d]...", e.cfg.Network.Driver, e.cfg.Network.ChainID)

	// Pre-flight approval check
	if e.cfg.Wallet.AutoApprove {
		clobRouter := e.cfg.Network.Contracts["clob_router"]
		collateral := e.cfg.Network.Contracts["collateral_token"]
		if clobRouter != "" && collateral != "" {
			logger.Infof("Verifying ERC-20 collateral allowance...")
			maxUint := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
			tx, err := e.adapter.EnsureAllowance(ctx, collateral, clobRouter, maxUint)
			if err != nil {
				logger.Warnf("EnsureAllowance pre-flight warning: %v", err)
			} else if tx != "" {
				logger.Infof("Allowance approved. Tx: %s", tx)
			}
		}
	}

	e.watcher.Subscribe(e.marketCh)
	go e.watcher.Start(ctx)
	go e.executor.StartWorkerPool(ctx, 4)
	go e.settler.Start(ctx)

	go e.evaluationLoop(ctx)

	return nil
}

func (e *Engine) Stop() {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	logger.Infof("Beaverish Core Engine stopped.")
}

func (e *Engine) evaluationLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-e.marketCh:
			now := time.Now().Unix()
			timeRemaining := snapshot.ExpiryTimestamp - now

			// Risk & Timing Pre-filter
			if timeRemaining <= int64(e.cfg.Risk.ExpiryCutoffSeconds) {
				logger.Debugf("Market %s rejected: within expiry cutoff buffer (%ds <= %ds)",
					snapshot.MarketID, timeRemaining, e.cfg.Risk.ExpiryCutoffSeconds)
				continue
			}

			// Evaluate strategies
			for _, strat := range e.strategies {
				sig := strat.Evaluate(snapshot)
				if sig.ShouldExecute {
					logger.Infof("Strategy [%s] generated positive signal for market %s", strat.Name(), snapshot.MarketID)
					e.executor.Dispatch(sig)
					break
				}
			}
		}
	}
}

func (e *Engine) Adapter() types.MarketAdapter {
	return e.adapter
}

func (e *Engine) Settler() *SettlementDaemon {
	return e.settler
}
