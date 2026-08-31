package strategy

import (
	"math/big"

	"github.com/nathfavour/beaverish/pkg/types"
)

type MomentumStrategy struct {
	name            string
	maxBetSizeUnits float64
}

func NewMomentumStrategy(maxBetSize float64) *MomentumStrategy {
	return &MomentumStrategy{
		name:            "MomentumDivergence",
		maxBetSizeUnits: maxBetSize,
	}
}

func (s *MomentumStrategy) Name() string {
	return s.name
}

func (s *MomentumStrategy) Evaluate(snapshot types.MarketSnapshot) types.Signal {
	if snapshot.Status != types.MarketStatusActive {
		return types.Signal{ShouldExecute: false, MarketID: snapshot.MarketID, Reason: "Market is not active"}
	}

	// Reference fast feed evaluation
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	betSizeFloat := new(big.Float).Mul(big.NewFloat(s.maxBetSizeUnits), new(big.Float).SetInt(oneUnit))
	betSizeWei, _ := betSizeFloat.Int(nil)

	// If Up ask is very cheap (< 0.40) and Down ask is expensive (> 0.65), follow trend
	if snapshot.BestAskUp != nil && snapshot.BestAskDown != nil {
		p40 := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(40)), big.NewInt(100))
		p65 := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(65)), big.NewInt(100))

		if snapshot.BestAskDown.Cmp(p40) < 0 && snapshot.BestAskUp.Cmp(p65) > 0 {
			return types.Signal{
				ShouldExecute: true,
				MarketID:      snapshot.MarketID,
				TargetSide:    types.OutcomeDown,
				TargetPrice:   snapshot.BestAskDown,
				PositionSize:  betSizeWei,
				Reason:        "Momentum divergence favor DOWN",
			}
		}

		if snapshot.BestAskUp.Cmp(p40) < 0 && snapshot.BestAskDown.Cmp(p65) > 0 {
			return types.Signal{
				ShouldExecute: true,
				MarketID:      snapshot.MarketID,
				TargetSide:    types.OutcomeUp,
				TargetPrice:   snapshot.BestAskUp,
				PositionSize:  betSizeWei,
				Reason:        "Momentum divergence favor UP",
			}
		}
	}

	return types.Signal{
		ShouldExecute: false,
		MarketID:      snapshot.MarketID,
		Reason:        "No momentum signal",
	}
}
