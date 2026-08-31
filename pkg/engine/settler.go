package engine

import (
	"context"
	"time"

	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/types"
)

type SettlementDaemon struct {
	adapter      types.MarketAdapter
	sweepInterval time.Duration
}

func NewSettlementDaemon(adapter types.MarketAdapter, interval time.Duration) *SettlementDaemon {
	return &SettlementDaemon{
		adapter:      adapter,
		sweepInterval: interval,
	}
}

func (sd *SettlementDaemon) Start(ctx context.Context) {
	ticker := time.NewTicker(sd.sweepInterval)
	defer ticker.Stop()

	logger.Infof("SettlementDaemon started with interval: %v", sd.sweepInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Infof("SettlementDaemon stopping...")
			return
		case <-ticker.C:
			sd.Sweep(ctx)
		}
	}
}

func (sd *SettlementDaemon) Sweep(ctx context.Context) ([]string, error) {
	markets, err := sd.adapter.GetActiveMarkets(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	var claimedTxs []string

	for _, m := range markets {
		// If market expired or locked/resolved, claim
		if m.ExpiryTimestamp <= now || m.Status == types.MarketStatusResolved {
			logger.Infof("Sweep: market %s reached expiration/resolved. Attempting settlement payout claim...", m.MarketID)
			tx, err := sd.adapter.ClaimWinningPayout(ctx, m.MarketID)
			if err != nil {
				logger.Debugf("Claim payout for %s: %v", m.MarketID, err)
			} else if tx != "" {
				logger.Infof("🎉 Payout claimed for %s! Tx: %s", m.MarketID, tx)
				claimedTxs = append(claimedTxs, tx)
			}
		}
	}

	return claimedTxs, nil
}
