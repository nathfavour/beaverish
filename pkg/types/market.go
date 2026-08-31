package types

import (
	"math/big"
	"time"
)

type OutcomeSide uint8

const (
	OutcomeUp OutcomeSide = iota
	OutcomeDown
)

func (s OutcomeSide) String() string {
	switch s {
	case OutcomeUp:
		return "UP"
	case OutcomeDown:
		return "DOWN"
	default:
		return "UNKNOWN"
	}
}

func ParseOutcomeSide(sideStr string) (OutcomeSide, bool) {
	switch sideStr {
	case "UP", "up", "0":
		return OutcomeUp, true
	case "DOWN", "down", "1":
		return OutcomeDown, true
	default:
		return OutcomeUp, false
	}
}

type MarketStatus uint8

const (
	MarketStatusActive MarketStatus = iota
	MarketStatusLocked
	MarketStatusResolved
)

func (s MarketStatus) String() string {
	switch s {
	case MarketStatusActive:
		return "ACTIVE"
	case MarketStatusLocked:
		return "LOCKED"
	case MarketStatusResolved:
		return "RESOLVED"
	default:
		return "UNKNOWN"
	}
}

type MarketSnapshot struct {
	MarketID        string       `json:"market_id"`
	ContractAddress string       `json:"contract_address"`
	UnderlyingAsset string       `json:"underlying_asset"`
	StrikePrice     *big.Int     `json:"strike_price"`
	ExpiryTimestamp int64        `json:"expiry_timestamp"`
	LockTimestamp   int64        `json:"lock_timestamp"`
	BestBidUp       *big.Int     `json:"best_bid_up"`
	BestAskUp       *big.Int     `json:"best_ask_up"`
	BestBidDown     *big.Int     `json:"best_bid_down"`
	BestAskDown     *big.Int     `json:"best_ask_down"`
	Status          MarketStatus `json:"status"`
	WinningOutcome  OutcomeSide  `json:"winning_outcome"`
}

type OrderType uint8

const (
	OrderTypeLimit OrderType = iota
	OrderTypeMarket
)

type Order struct {
	ID            string      `json:"id"`
	MarketID      string      `json:"market_id"`
	Side          OutcomeSide `json:"side"`
	Type          OrderType   `json:"type"`
	Price         *big.Int    `json:"price"`
	Amount        *big.Int    `json:"amount"`
	TxHash        string      `json:"tx_hash"`
	CreatedAt     time.Time   `json:"created_at"`
	IsSettled     bool        `json:"is_settled"`
	SettledTxHash string      `json:"settled_tx_hash,omitempty"`
}

type Position struct {
	MarketID     string      `json:"market_id"`
	Side         OutcomeSide `json:"side"`
	Amount       *big.Int    `json:"amount"`
	AveragePrice *big.Int    `json:"average_price"`
	TxHash       string      `json:"tx_hash"`
	CreatedAt    time.Time   `json:"created_at"`
	IsSettled    bool        `json:"is_settled"`
}

type AccountStatus struct {
	Address           string   `json:"address"`
	NativeBalance     *big.Int `json:"native_balance"`
	CollateralBalance *big.Int `json:"collateral_balance"`
	OpenPositions     int      `json:"open_positions"`
	RealizedPnL       *big.Int `json:"realized_pnl"`
}
