package types

import "math/big"

type OrderbookLevel struct {
	Price  *big.Int `json:"price"`
	Amount *big.Int `json:"amount"`
}

type OrderbookDepth struct {
	MarketID string           `json:"market_id"`
	BidsUp   []OrderbookLevel `json:"bids_up"`
	AsksUp   []OrderbookLevel `json:"asks_up"`
	BidsDown []OrderbookLevel `json:"bids_down"`
	AsksDown []OrderbookLevel `json:"asks_down"`
}
