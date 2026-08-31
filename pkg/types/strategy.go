package types

import (
	"math/big"
)

type Signal struct {
	ShouldExecute bool        `json:"should_execute"`
	MarketID      string      `json:"market_id"`
	TargetSide    OutcomeSide `json:"target_side"`
	TargetPrice   *big.Int    `json:"target_price"`
	PositionSize  *big.Int    `json:"position_size"`
	Reason        string      `json:"reason"`
}

type Strategy interface {
	Name() string
	Evaluate(snapshot MarketSnapshot) Signal
}
