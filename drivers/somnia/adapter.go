package somnia

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/pkg/engine"
	"github.com/nathfavour/beaverish/pkg/logger"
	"github.com/nathfavour/beaverish/pkg/signer"
	"github.com/nathfavour/beaverish/pkg/types"
)

var (
	// Method signatures for DreamDEX CLOB / Event Contracts
	placeLimitOrderMethodID  = crypto.Keccak256([]byte("placeLimitOrder(bytes32,uint8,uint256,uint256)"))[:4]
	placeMarketOrderMethodID = crypto.Keccak256([]byte("placeMarketOrder(bytes32,uint8,uint256)"))[:4]
	cancelOrderMethodID      = crypto.Keccak256([]byte("cancelOrder(bytes32)"))[:4]
	claimPayoutMethodID      = crypto.Keccak256([]byte("claimPayout(bytes32)"))[:4]
	getActiveMarketsMethodID = crypto.Keccak256([]byte("getActiveMarkets()"))[:4]
)

type SomniaAdapter struct {
	mu           sync.RWMutex
	cfg          *config.Config
	client       *Client
	signer       *signer.Signer
	nonceManager *engine.NonceManager

	// Local mock / cache for offline/dry-run and active state
	mockMarkets   map[string]types.MarketSnapshot
	openPositions []types.Position
	realizedPnL   *big.Int
}

func NewSomniaAdapter(cfg *config.Config, s *signer.Signer, nonceMgr *engine.NonceManager) (*SomniaAdapter, error) {
	client, err := Dial(cfg.Network.RPCURL, cfg.Network.WSURL)
	if err != nil {
		logger.Warnf("Somnia client dial warning: %v", err)
	}

	adapter := &SomniaAdapter{
		cfg:           cfg,
		client:        client,
		signer:        s,
		nonceManager:  nonceMgr,
		mockMarkets:   make(map[string]types.MarketSnapshot),
		openPositions: make([]types.Position, 0),
		realizedPnL:   big.NewInt(0),
	}

	adapter.initDefaultMarkets()
	return adapter, nil
}

func (a *SomniaAdapter) initDefaultMarkets() {
	now := time.Now().Unix()
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	// Sample Somnia DreamDEX BTC 5m Event Market
	pAskUp := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(48)), big.NewInt(100))
	pBidUp := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(47)), big.NewInt(100))
	pAskDown := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(47)), big.NewInt(100))
	pBidDown := new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(46)), big.NewInt(100))

	m1 := types.MarketSnapshot{
		MarketID:        "0x7b1c3a8e9d0f4125a83b27e891c3f5e042a9b1c7000000000000000000000001",
		ContractAddress: "0x1111111111111111111111111111111111111111",
		UnderlyingAsset: "BTC",
		StrikePrice:     big.NewInt(92500000000), // $92,500.00
		ExpiryTimestamp: now + 300,
		LockTimestamp:   now + 240,
		BestBidUp:       pBidUp,
		BestAskUp:       pAskUp,
		BestBidDown:     pBidDown,
		BestAskDown:     pAskDown,
		Status:          types.MarketStatusActive,
	}

	m2 := types.MarketSnapshot{
		MarketID:        "0x9e8a7b6c5d4e3f21a0b9c8d7e6f5a4b3c2d1e0f9000000000000000000000002",
		ContractAddress: "0x2222222222222222222222222222222222222222",
		UnderlyingAsset: "ETH",
		StrikePrice:     big.NewInt(2650000000), // $2,650.00
		ExpiryTimestamp: now + 600,
		LockTimestamp:   now + 540,
		BestBidUp:       new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(52)), big.NewInt(100)),
		BestAskUp:       new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(54)), big.NewInt(100)),
		BestBidDown:     new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(44)), big.NewInt(100)),
		BestAskDown:     new(big.Int).Div(new(big.Int).Mul(oneUnit, big.NewInt(46)), big.NewInt(100)),
		Status:          types.MarketStatusActive,
	}

	a.mockMarkets[m1.MarketID] = m1
	a.mockMarkets[m2.MarketID] = m2
}

func (a *SomniaAdapter) Name() string {
	return "somnia"
}

func (a *SomniaAdapter) GetActiveMarkets(ctx context.Context) ([]types.MarketSnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	res := make([]types.MarketSnapshot, 0, len(a.mockMarkets))
	for _, m := range a.mockMarkets {
		res = append(res, m)
	}
	return res, nil
}

func (a *SomniaAdapter) GetMarketSnapshot(ctx context.Context, marketID string) (*types.MarketSnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	m, ok := a.mockMarkets[marketID]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", marketID)
	}
	return &m, nil
}

func (a *SomniaAdapter) PlaceLimitOrder(
	ctx context.Context,
	marketID string,
	side types.OutcomeSide,
	price *big.Int,
	amount *big.Int,
) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg.Runtime.DryRun || a.client.RPCClient == nil || a.signer == nil {
		txHash := fmt.Sprintf("0xmock_limit_%x", time.Now().UnixNano())
		logger.Infof("[DRY-RUN/MOCK] PlaceLimitOrder market=%s side=%s price=%s amount=%s tx=%s",
			marketID, side.String(), price.String(), amount.String(), txHash)
		a.openPositions = append(a.openPositions, types.Position{
			MarketID:     marketID,
			Side:         side,
			Amount:       amount,
			AveragePrice: price,
			TxHash:       txHash,
			CreatedAt:    time.Now(),
			IsSettled:    false,
		})
		return txHash, nil
	}

	clobAddr := common.HexToAddress(a.cfg.Network.Contracts["clob_router"])
	marketIDBytes := common.HexToHash(marketID).Bytes()

	var data []byte
	data = append(data, placeLimitOrderMethodID...)
	data = append(data, marketIDBytes...)
	data = append(data, common.LeftPadBytes([]byte{byte(side)}, 32)...)
	data = append(data, common.LeftPadBytes(price.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)

	nonce, err := a.nonceManager.NextNonce(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain nonce: %w", err)
	}

	signedTx, err := a.signer.BuildAndSignTx(ctx, a.client.RPCClient, clobAddr, big.NewInt(0), data, nonce, 250000)
	if err != nil {
		return "", fmt.Errorf("failed to sign limit order: %w", err)
	}

	if err := a.client.RPCClient.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("failed to send limit order tx: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	a.openPositions = append(a.openPositions, types.Position{
		MarketID:     marketID,
		Side:         side,
		Amount:       amount,
		AveragePrice: price,
		TxHash:       txHash,
		CreatedAt:    time.Now(),
		IsSettled:    false,
	})
	return txHash, nil
}

func (a *SomniaAdapter) PlaceMarketOrder(
	ctx context.Context,
	marketID string,
	side types.OutcomeSide,
	amount *big.Int,
) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg.Runtime.DryRun || a.client.RPCClient == nil || a.signer == nil {
		txHash := fmt.Sprintf("0xmock_market_%x", time.Now().UnixNano())
		logger.Infof("[DRY-RUN/MOCK] PlaceMarketOrder market=%s side=%s amount=%s tx=%s",
			marketID, side.String(), amount.String(), txHash)
		a.openPositions = append(a.openPositions, types.Position{
			MarketID:     marketID,
			Side:         side,
			Amount:       amount,
			AveragePrice: big.NewInt(0),
			TxHash:       txHash,
			CreatedAt:    time.Now(),
			IsSettled:    false,
		})
		return txHash, nil
	}

	clobAddr := common.HexToAddress(a.cfg.Network.Contracts["clob_router"])
	marketIDBytes := common.HexToHash(marketID).Bytes()

	var data []byte
	data = append(data, placeMarketOrderMethodID...)
	data = append(data, marketIDBytes...)
	data = append(data, common.LeftPadBytes([]byte{byte(side)}, 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)

	nonce, err := a.nonceManager.NextNonce(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain nonce: %w", err)
	}

	signedTx, err := a.signer.BuildAndSignTx(ctx, a.client.RPCClient, clobAddr, big.NewInt(0), data, nonce, 250000)
	if err != nil {
		return "", fmt.Errorf("failed to sign market order: %w", err)
	}

	if err := a.client.RPCClient.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("failed to send market order tx: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	a.openPositions = append(a.openPositions, types.Position{
		MarketID:     marketID,
		Side:         side,
		Amount:       amount,
		AveragePrice: big.NewInt(0),
		TxHash:       txHash,
		CreatedAt:    time.Now(),
		IsSettled:    false,
	})
	return txHash, nil
}

func (a *SomniaAdapter) CancelOrder(ctx context.Context, orderID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg.Runtime.DryRun || a.client.RPCClient == nil || a.signer == nil {
		txHash := fmt.Sprintf("0xmock_cancel_%x", time.Now().UnixNano())
		logger.Infof("[DRY-RUN/MOCK] CancelOrder orderID=%s tx=%s", orderID, txHash)
		return txHash, nil
	}

	clobAddr := common.HexToAddress(a.cfg.Network.Contracts["clob_router"])
	orderIDBytes := common.HexToHash(orderID).Bytes()

	var data []byte
	data = append(data, cancelOrderMethodID...)
	data = append(data, orderIDBytes...)

	nonce, err := a.nonceManager.NextNonce(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain nonce: %w", err)
	}

	signedTx, err := a.signer.BuildAndSignTx(ctx, a.client.RPCClient, clobAddr, big.NewInt(0), data, nonce, 150000)
	if err != nil {
		return "", fmt.Errorf("failed to sign cancel order: %w", err)
	}

	if err := a.client.RPCClient.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("failed to send cancel order tx: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

func (a *SomniaAdapter) ClaimWinningPayout(ctx context.Context, marketID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg.Runtime.DryRun || a.client.RPCClient == nil || a.signer == nil {
		txHash := fmt.Sprintf("0xmock_claim_%x", time.Now().UnixNano())
		logger.Infof("[DRY-RUN/MOCK] ClaimWinningPayout market=%s tx=%s", marketID, txHash)
		for i := range a.openPositions {
			if a.openPositions[i].MarketID == marketID {
				a.openPositions[i].IsSettled = true
			}
		}
		return txHash, nil
	}

	clobAddr := common.HexToAddress(a.cfg.Network.Contracts["clob_router"])
	marketIDBytes := common.HexToHash(marketID).Bytes()

	var data []byte
	data = append(data, claimPayoutMethodID...)
	data = append(data, marketIDBytes...)

	nonce, err := a.nonceManager.NextNonce(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain nonce: %w", err)
	}

	signedTx, err := a.signer.BuildAndSignTx(ctx, a.client.RPCClient, clobAddr, big.NewInt(0), data, nonce, 180000)
	if err != nil {
		return "", fmt.Errorf("failed to sign claim transaction: %w", err)
	}

	if err := a.client.RPCClient.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("failed to send claim transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	for i := range a.openPositions {
		if a.openPositions[i].MarketID == marketID {
			a.openPositions[i].IsSettled = true
		}
	}
	return txHash, nil
}

func (a *SomniaAdapter) EnsureAllowance(
	ctx context.Context,
	tokenAddress string,
	spenderAddress string,
	minAmount *big.Int,
) (string, error) {
	if a.signer == nil || a.client.RPCClient == nil || a.cfg.Runtime.DryRun {
		return "", nil
	}

	tokenAddr := common.HexToAddress(tokenAddress)
	spenderAddr := common.HexToAddress(spenderAddress)
	if tokenAddr == (common.Address{}) || spenderAddr == (common.Address{}) {
		return "", nil
	}

	nonce, err := a.nonceManager.NextNonce(ctx)
	if err != nil {
		return "", err
	}

	return signer.EnsureAllowance(ctx, a.client.RPCClient, a.signer, tokenAddr, spenderAddr, minAmount, nonce)
}

func (a *SomniaAdapter) GetAccountStatus(ctx context.Context) (*types.AccountStatus, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	addrStr := "0x0000000000000000000000000000000000000000"
	var nativeBal *big.Int = big.NewInt(0)
	var colBal *big.Int = big.NewInt(0)

	if a.signer != nil {
		addrStr = a.signer.Address().Hex()
		if a.client.RPCClient != nil {
			bal, err := a.client.RPCClient.BalanceAt(ctx, a.signer.Address(), nil)
			if err == nil {
				nativeBal = bal
			}
			colAddr := common.HexToAddress(a.cfg.Network.Contracts["collateral_token"])
			if colAddr != (common.Address{}) {
				cbal, err := signer.CheckTokenBalance(ctx, a.client.RPCClient, colAddr, a.signer.Address())
				if err == nil {
					colBal = cbal
				}
			}
		}
	}

	activePositionsCount := 0
	for _, p := range a.openPositions {
		if !p.IsSettled {
			activePositionsCount++
		}
	}

	return &types.AccountStatus{
		Address:           addrStr,
		NativeBalance:     nativeBal,
		CollateralBalance: colBal,
		OpenPositions:     activePositionsCount,
		RealizedPnL:       a.realizedPnL,
	}, nil
}

func (a *SomniaAdapter) GetPositions(ctx context.Context) ([]types.Position, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	cp := make([]types.Position, len(a.openPositions))
	copy(cp, a.openPositions)
	return cp, nil
}

func (a *SomniaAdapter) GetOrderbook(ctx context.Context, marketID string) (*types.OrderbookDepth, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	m, ok := a.mockMarkets[marketID]
	if !ok {
		return nil, fmt.Errorf("market not found: %s", marketID)
	}

	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	vol := new(big.Int).Mul(oneUnit, big.NewInt(100)) // 100 units volume

	return &types.OrderbookDepth{
		MarketID: marketID,
		BidsUp:   []types.OrderbookLevel{{Price: m.BestBidUp, Amount: vol}},
		AsksUp:   []types.OrderbookLevel{{Price: m.BestAskUp, Amount: vol}},
		BidsDown: []types.OrderbookLevel{{Price: m.BestBidDown, Amount: vol}},
		AsksDown: []types.OrderbookLevel{{Price: m.BestAskDown, Amount: vol}},
	}, nil
}
