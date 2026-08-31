package somnia

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nathfavour/beaverish/pkg/logger"
)

type Client struct {
	RPCClient *ethclient.Client
	rpcURL    string
	wsURL     string
}

func Dial(rpcURL, wsURL string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var client *ethclient.Client
	var err error

	if rpcURL != "" {
		client, err = ethclient.DialContext(ctx, rpcURL)
		if err != nil {
			logger.Warnf("Direct RPC dial error (%s): %v. Initializing with offline fallback capability.", rpcURL, err)
		}
	}

	return &Client{
		RPCClient: client,
		rpcURL:    rpcURL,
		wsURL:     wsURL,
	}, nil
}

func (c *Client) Close() {
	if c.RPCClient != nil {
		c.RPCClient.Close()
	}
}

func (c *Client) CallWithRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(100*(i+1)) * time.Millisecond):
		}
	}
	return fmt.Errorf("exceeded max retries: %w", err)
}
