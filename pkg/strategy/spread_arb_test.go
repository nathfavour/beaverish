package strategy

import (
	"math/big"
	"testing"

	"github.com/nathfavour/beaverish/pkg/types"
)

func TestSpreadArbStrategy_Evaluate(t *testing.T) {
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	strat := NewSpreadArbStrategy(0.04, 5.0)

	// Case 1: Arbitrage edge exists: AskUp = 0.45, AskDown = 0.45 (Sum = 0.90 < 0.96)
	p45 := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(45)), big.NewInt(100))
	snapshotEdge := types.MarketSnapshot{
		MarketID:    "test-market-1",
		BestAskUp:   p45,
		BestAskDown: p45,
		Status:      types.MarketStatusActive,
	}

	sig := strat.Evaluate(snapshotEdge)
	if !sig.ShouldExecute {
		t.Fatalf("Expected signal ShouldExecute to be true, got false. Reason: %s", sig.Reason)
	}

	// Case 2: No arbitrage edge: AskUp = 0.52, AskDown = 0.50 (Sum = 1.02 > 0.96)
	p52 := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(52)), big.NewInt(100))
	p50 := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(50)), big.NewInt(100))
	snapshotNoEdge := types.MarketSnapshot{
		MarketID:    "test-market-2",
		BestAskUp:   p52,
		BestAskDown: p50,
		Status:      types.MarketStatusActive,
	}

	sig2 := strat.Evaluate(snapshotNoEdge)
	if sig2.ShouldExecute {
		t.Fatalf("Expected signal ShouldExecute to be false for no edge")
	}
}
