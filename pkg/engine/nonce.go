package engine

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nathfavour/beaverish/pkg/logger"
)

type NonceManager struct {
	mu      sync.Mutex
	client  *ethclient.Client
	address common.Address
	nonce   uint64
	synced  bool
}

func NewNonceManager(client *ethclient.Client, address common.Address) *NonceManager {
	return &NonceManager{
		client:  client,
		address: address,
	}
}

func (nm *NonceManager) Sync(ctx context.Context) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.client == nil {
		nm.nonce = 0
		nm.synced = true
		return nil
	}

	nonce, err := nm.client.PendingNonceAt(ctx, nm.address)
	if err != nil {
		return err
	}
	nm.nonce = nonce
	nm.synced = true
	logger.Debugf("Nonce synced for %s: %d", nm.address.Hex(), nm.nonce)
	return nil
}

func (nm *NonceManager) NextNonce(ctx context.Context) (uint64, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.synced {
		if err := nm.syncLocked(ctx); err != nil {
			return 0, err
		}
	}

	curr := nm.nonce
	nm.nonce++
	return curr, nil
}

func (nm *NonceManager) Resync(ctx context.Context) error {
	return nm.Sync(ctx)
}

func (nm *NonceManager) syncLocked(ctx context.Context) error {
	if nm.client == nil {
		nm.nonce = 0
		nm.synced = true
		return nil
	}
	nonce, err := nm.client.PendingNonceAt(ctx, nm.address)
	if err != nil {
		return err
	}
	nm.nonce = nonce
	nm.synced = true
	return nil
}
