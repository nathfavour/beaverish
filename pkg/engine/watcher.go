package engine

import (
	"context"
	"sync"
	"time"

	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/types"
)

type MarketWatcher struct {
	adapter      types.MarketAdapter
	pollInterval time.Duration
	subscribers  []chan<- types.MarketSnapshot
	mu           sync.RWMutex
}

func NewMarketWatcher(adapter types.MarketAdapter, pollInterval time.Duration) *MarketWatcher {
	return &MarketWatcher{
		adapter:      adapter,
		pollInterval: pollInterval,
		subscribers:  make([]chan<- types.MarketSnapshot, 0),
	}
}

func (w *MarketWatcher) Subscribe(ch chan<- types.MarketSnapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subscribers = append(w.subscribers, ch)
}

func (w *MarketWatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	logger.Infof("MarketWatcher started with polling interval: %v", w.pollInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Infof("MarketWatcher stopping...")
			return
		case <-ticker.C:
			markets, err := w.adapter.GetActiveMarkets(ctx)
			if err != nil {
				logger.Warnf("Failed to poll active markets: %v", err)
				continue
			}

			w.mu.RLock()
			for _, m := range markets {
				for _, sub := range w.subscribers {
					select {
					case sub <- m:
					default:
						// Non-blocking if consumer buffer full
					}
				}
			}
			w.mu.RUnlock()
		}
	}
}
