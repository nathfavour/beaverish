package types

import (
	"context"
	"math/big"
)

type MarketAdapter interface {
	Name() string
	GetActiveMarkets(ctx context.Context) ([]MarketSnapshot, error)
	GetMarketSnapshot(ctx context.Context, marketID string) (*MarketSnapshot, error)
	PlaceLimitOrder(ctx context.Context, marketID string, side OutcomeSide, price *big.Int, amount *big.Int) (string, error)
	PlaceMarketOrder(ctx context.Context, marketID string, side OutcomeSide, amount *big.Int) (string, error)
	CancelOrder(ctx context.Context, orderID string) (string, error)
	ClaimWinningPayout(ctx context.Context, marketID string) (string, error)
	EnsureAllowance(ctx context.Context, tokenAddress string, spenderAddress string, minAmount *big.Int) (string, error)
	GetAccountStatus(ctx context.Context) (*AccountStatus, error)
	GetPositions(ctx context.Context) ([]Position, error)
	GetOrderbook(ctx context.Context, marketID string) (*OrderbookDepth, error)
}
