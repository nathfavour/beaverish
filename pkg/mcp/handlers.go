package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/pkg/engine"
	"github.com/nathfavour/beaverish/pkg/strategy"
	"github.com/nathfavour/beaverish/pkg/types"
)

type Handler struct {
	cfg        *config.Config
	eng        *engine.Engine
	strategies []types.Strategy
}

func NewHandler(cfg *config.Config, eng *engine.Engine) *Handler {
	return &Handler{
		cfg: cfg,
		eng: eng,
		strategies: []types.Strategy{
			strategy.NewSpreadArbStrategy(cfg.Risk.MinEdgeThreshold, cfg.Risk.MaxBetSizeUnits),
			strategy.NewMomentumStrategy(cfg.Risk.MaxBetSizeUnits),
		},
	}
}

func (h *Handler) GetTools() []Tool {
	return []Tool{
		{
			Name:        "get_markets",
			Description: "Retrieves all active binary event contracts on Somnia with strike prices, expirations, and current bid/ask depths.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"underlying": {
						Type:        "string",
						Description: "Optional underlying asset filter (e.g. BTC, ETH)",
					},
				},
			},
		},
		{
			Name:        "evaluate_market",
			Description: "Runs deterministic spread and probability evaluation on a target market to check for mispricing and parity arbitrage edge.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"market_id": {
						Type:        "string",
						Description: "The hex ID or address of the binary event market to evaluate",
					},
				},
				Required: []string{"market_id"},
			},
		},
		{
			Name:        "execute_order",
			Description: "Signs and submits an order to buy binary outcome shares (UP or DOWN) on Somnia testnet.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"market_id": {
						Type:        "string",
						Description: "The ID of the market to trade",
					},
					"side": {
						Type:        "string",
						Description: "Outcome side: UP or DOWN",
						Enum:        []string{"UP", "DOWN"},
					},
					"amount": {
						Type:        "number",
						Description: "Position amount in standard units (e.g. 1.0, 5.0)",
					},
					"price_limit": {
						Type:        "number",
						Description: "Optional maximum price (e.g. 0.48). If omitted or 0, executes as market order",
					},
				},
				Required: []string{"market_id", "side", "amount"},
			},
		},
		{
			Name:        "sweep_settlements",
			Description: "Inspects open positions, checks for resolved winning contracts, and executes payout claims.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "get_account_status",
			Description: "Returns testnet gas token balance, collateral balance, open positions count, and realized PnL.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
	}
}

func (h *Handler) CallTool(ctx context.Context, name string, args map[string]interface{}) (ToolResult, error) {
	switch name {
	case "get_markets":
		return h.handleGetMarkets(ctx, args)
	case "evaluate_market":
		return h.handleEvaluateMarket(ctx, args)
	case "execute_order":
		return h.handleExecuteOrder(ctx, args)
	case "sweep_settlements":
		return h.handleSweepSettlements(ctx, args)
	case "get_account_status":
		return h.handleGetAccountStatus(ctx, args)
	default:
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", name)}},
			IsError: true,
		}, fmt.Errorf("unknown tool: %s", name)
	}
}

func (h *Handler) handleGetMarkets(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	markets, err := h.eng.Adapter().GetActiveMarkets(ctx)
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Error querying markets: %v", err)}},
			IsError: true,
		}, nil
	}

	filterUnderlying, _ := args["underlying"].(string)
	var filtered []types.MarketSnapshot
	for _, m := range markets {
		if filterUnderlying == "" || strings.EqualFold(m.UnderlyingAsset, filterUnderlying) {
			filtered = append(filtered, m)
		}
	}

	data, _ := json.MarshalIndent(filtered, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (h *Handler) handleEvaluateMarket(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	marketID, ok := args["market_id"].(string)
	if !ok || marketID == "" {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: "market_id is required"}},
			IsError: true,
		}, nil
	}

	snapshot, err := h.eng.Adapter().GetMarketSnapshot(ctx, marketID)
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Market not found: %v", err)}},
			IsError: true,
		}, nil
	}

	type EvalResult struct {
		MarketID string         `json:"market_id"`
		Signals  []types.Signal `json:"signals"`
	}

	res := EvalResult{MarketID: marketID, Signals: make([]types.Signal, 0)}
	for _, strat := range h.strategies {
		res.Signals = append(res.Signals, strat.Evaluate(*snapshot))
	}

	data, _ := json.MarshalIndent(res, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (h *Handler) handleExecuteOrder(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	marketID, _ := args["market_id"].(string)
	sideStr, _ := args["side"].(string)
	
	side, valid := types.ParseOutcomeSide(sideStr)
	if !valid {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Invalid side: %s. Must be UP or DOWN", sideStr)}},
			IsError: true,
		}, nil
	}

	var amountFloat float64
	switch v := args["amount"].(type) {
	case float64:
		amountFloat = v
	case string:
		amountFloat, _ = strconv.ParseFloat(v, 64)
	}

	if amountFloat <= 0 {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: "amount must be greater than 0"}},
			IsError: true,
		}, nil
	}

	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	amountWei, _ := new(big.Float).Mul(big.NewFloat(amountFloat), new(big.Float).SetInt(oneUnit)).Int(nil)

	var priceFloat float64
	switch v := args["price_limit"].(type) {
	case float64:
		priceFloat = v
	case string:
		priceFloat, _ = strconv.ParseFloat(v, 64)
	}

	var txHash string
	var err error

	if priceFloat > 0 {
		priceWei, _ := new(big.Float).Mul(big.NewFloat(priceFloat), new(big.Float).SetInt(oneUnit)).Int(nil)
		txHash, err = h.eng.Adapter().PlaceLimitOrder(ctx, marketID, side, priceWei, amountWei)
	} else {
		txHash, err = h.eng.Adapter().PlaceMarketOrder(ctx, marketID, side, amountWei)
	}

	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Execution failed: %v", err)}},
			IsError: true,
		}, nil
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Order placed successfully. Transaction Hash: %s", txHash)}},
	}, nil
}

func (h *Handler) handleSweepSettlements(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	txs, err := h.eng.Settler().Sweep(ctx)
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Sweep error: %v", err)}},
			IsError: true,
		}, nil
	}

	if len(txs) == 0 {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: "No open positions eligible for settlement payout sweep."}},
		}, nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"claimed_transactions": txs,
		"count":                len(txs),
	}, "", "  ")

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (h *Handler) handleGetAccountStatus(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	status, err := h.eng.Adapter().GetAccountStatus(ctx)
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Error retrieving account status: %v", err)}},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}
