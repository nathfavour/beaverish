package strategy

import (
	"math/big"

	"github.com/nathfavour/beaverish/pkg/types"
)

type SpreadArbStrategy struct {
	name             string
	minEdgeThreshold float64
	maxBetSizeUnits  float64
}

func NewSpreadArbStrategy(minEdge float64, maxBetSize float64) *SpreadArbStrategy {
	return &SpreadArbStrategy{
		name:             "SpreadArb",
		minEdgeThreshold: minEdge,
		maxBetSizeUnits:  maxBetSize,
	}
}

func (s *SpreadArbStrategy) Name() string {
	return s.name
}

// Evaluate checks if Ask(UP) + Ask(DOWN) < 1.0 - EdgeThreshold (Parity Edge)
// Or if Bid-Ask spread allows profitable arb.
func (s *SpreadArbStrategy) Evaluate(snapshot types.MarketSnapshot) types.Signal {
	if snapshot.Status != types.MarketStatusActive {
		return types.Signal{ShouldExecute: false, MarketID: snapshot.MarketID, Reason: "Market is not active"}
	}

	if snapshot.BestAskUp == nil || snapshot.BestAskDown == nil {
		return types.Signal{ShouldExecute: false, MarketID: snapshot.MarketID, Reason: "Missing orderbook asks"}
	}

	// 1.0 unit in wei scale (1e18)
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	sumAsks := new(big.Int).Add(snapshot.BestAskUp, snapshot.BestAskDown)

	// Edge calculation in 1e18 scale: edge = 1e18 - sumAsks
	edge := new(big.Int).Sub(oneUnit, sumAsks)

	// Threshold in 1e18 scale
	thresholdFloat := big.NewFloat(s.minEdgeThreshold)
	oneUnitFloat := new(big.Float).SetInt(oneUnit)
	thresholdUnits := new(big.Float).Mul(thresholdFloat, oneUnitFloat)
	thresholdWei, _ := thresholdUnits.Int(nil)

	if edge.Cmp(thresholdWei) > 0 {
		// Parity arbitrage opportunity detected: buy underpriced side (or both)
		// Pick the cheaper side to fill
		targetSide := types.OutcomeUp
		targetPrice := snapshot.BestAskUp
		if snapshot.BestAskDown.Cmp(snapshot.BestAskUp) < 0 {
			targetSide = types.OutcomeDown
			targetPrice = snapshot.BestAskDown
		}

		// Calculate bet size in wei
		betSizeFloat := new(big.Float).Mul(big.NewFloat(s.maxBetSizeUnits), oneUnitFloat)
		betSizeWei, _ := betSizeFloat.Int(nil)

		return types.Signal{
			ShouldExecute: true,
			MarketID:      snapshot.MarketID,
			TargetSide:    targetSide,
			TargetPrice:   targetPrice,
			PositionSize:  betSizeWei,
			Reason:        "Parity edge detected: Ask(UP)+Ask(DOWN) < 1.0 - EdgeThreshold",
		}
	}

	return types.Signal{
		ShouldExecute: false,
		MarketID:      snapshot.MarketID,
		Reason:        "No edge detected below threshold",
	}
}
