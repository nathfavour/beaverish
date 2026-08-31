package engine

import (
	"context"
	"math/big"
	"sync"

	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/types"
)

type ExecutionPipeline struct {
	mu           sync.Mutex
	cfg          *config.Config
	adapter      types.MarketAdapter
	pendingTasks chan types.Signal
	sessionLoss  *big.Int
	halted       bool
}

func NewExecutionPipeline(cfg *config.Config, adapter types.MarketAdapter, queueSize int) *ExecutionPipeline {
	return &ExecutionPipeline{
		cfg:          cfg,
		adapter:      adapter,
		pendingTasks: make(chan types.Signal, queueSize),
		sessionLoss:  big.NewInt(0),
		halted:       false,
	}
}

func (ep *ExecutionPipeline) Dispatch(sig types.Signal) {
	ep.mu.Lock()
	if ep.halted {
		ep.mu.Unlock()
		logger.Warnf("Execution skipped: Drawdown circuit breaker active!")
		return
	}
	ep.mu.Unlock()

	select {
	case ep.pendingTasks <- sig:
	default:
		logger.Warnf("Execution queue full, dropping signal for %s", sig.MarketID)
	}
}

func (ep *ExecutionPipeline) StartWorkerPool(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			logger.Debugf("Execution worker #%d spawned", workerID)
			for {
				select {
				case <-ctx.Done():
					return
				case sig := <-ep.pendingTasks:
					ep.executeSignal(ctx, sig)
				}
			}
		}(i)
	}
}

func (ep *ExecutionPipeline) executeSignal(ctx context.Context, sig types.Signal) {
	if !sig.ShouldExecute {
		return
	}

	logger.Infof("⚡ Executing Signal on Market [%s] Side [%s] Price [%s] Size [%s] Reason: %s",
		sig.MarketID, sig.TargetSide.String(), sig.TargetPrice.String(), sig.PositionSize.String(), sig.Reason)

	var txHash string
	var err error

	if sig.TargetPrice != nil && sig.TargetPrice.Sign() > 0 {
		txHash, err = ep.adapter.PlaceLimitOrder(ctx, sig.MarketID, sig.TargetSide, sig.TargetPrice, sig.PositionSize)
	} else {
		txHash, err = ep.adapter.PlaceMarketOrder(ctx, sig.MarketID, sig.TargetSide, sig.PositionSize)
	}

	if err != nil {
		logger.Errorf("Failed to execute order for market %s: %v", sig.MarketID, err)
		return
	}

	logger.Infof("✅ Order executed successfully! TxHash: %s", txHash)
}
