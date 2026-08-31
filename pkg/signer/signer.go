package signer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Signer struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    *big.Int
}

func NewSigner(hexKey string, chainID int64) (*Signer, error) {
	if hexKey == "" {
		return nil, errors.New("private key is empty")
	}
	cleanKey := strings.TrimPrefix(hexKey, "0x")
	privKey, err := crypto.HexToECDSA(cleanKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	pubKey := privKey.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot cast public key to ECDSA")
	}

	addr := crypto.PubkeyToAddress(*pubKeyECDSA)
	return &Signer{
		privateKey: privKey,
		address:    addr,
		chainID:    big.NewInt(chainID),
	}, nil
}

func (s *Signer) Address() common.Address {
	return s.address
}

func (s *Signer) ChainID() *big.Int {
	return s.chainID
}

func (s *Signer) SignTx(tx *types.Transaction) (*types.Transaction, error) {
	signer := types.NewLondonSigner(s.chainID)
	signedTx, err := types.SignTx(tx, signer, s.privateKey)
	if err != nil {
		// Fallback to EIP-155 signer if chain is legacy
		signerLegacy := types.NewEIP155Signer(s.chainID)
		return types.SignTx(tx, signerLegacy, s.privateKey)
	}
	return signedTx, nil
}

func (s *Signer) SignMessage(message []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(message)
	return crypto.Sign(hash.Bytes(), s.privateKey)
}

func (s *Signer) BuildAndSignTx(
	ctx context.Context,
	client *ethclient.Client,
	to common.Address,
	value *big.Int,
	data []byte,
	nonce uint64,
	gasLimit uint64,
) (*types.Transaction, error) {
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// Apply 25% safety overhead to gas price
	gasPriceWithBuffer := new(big.Int).Mul(gasPrice, big.NewInt(125))
	gasPriceWithBuffer.Div(gasPriceWithBuffer, big.NewInt(100))

	if gasLimit == 0 {
		gasLimit = 300000 // safe fallback limit
	}

	if value == nil {
		value = big.NewInt(0)
	}

	rawTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPriceWithBuffer,
		Data:     data,
	})

	return s.SignTx(rawTx)
}
