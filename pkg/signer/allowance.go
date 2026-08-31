package signer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nathfavour/beaverish/pkg/logger"
)

var (
	allowanceMethodID = crypto.Keccak256([]byte("allowance(address,address)"))[:4]
	approveMethodID   = crypto.Keccak256([]byte("approve(address,uint256)"))[:4]
	balanceOfMethodID = crypto.Keccak256([]byte("balanceOf(address)"))[:4]
)

func CheckAllowance(
	ctx context.Context,
	client *ethclient.Client,
	tokenAddr common.Address,
	ownerAddr common.Address,
	spenderAddr common.Address,
) (*big.Int, error) {
	if tokenAddr == (common.Address{}) || spenderAddr == (common.Address{}) {
		return big.NewInt(0), nil
	}

	var data []byte
	data = append(data, allowanceMethodID...)
	data = append(data, common.LeftPadBytes(ownerAddr.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(spenderAddr.Bytes(), 32)...)

	msg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}

	res, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call allowance: %w", err)
	}

	if len(res) < 32 {
		return big.NewInt(0), nil
	}

	allowance := new(big.Int).SetBytes(res[:32])
	return allowance, nil
}

func EnsureAllowance(
	ctx context.Context,
	client *ethclient.Client,
	s *Signer,
	tokenAddr common.Address,
	spenderAddr common.Address,
	requiredAmount *big.Int,
	nonce uint64,
) (string, error) {
	if tokenAddr == (common.Address{}) || spenderAddr == (common.Address{}) {
		logger.Debugf("Allowance skipped: null token or spender address")
		return "", nil
	}

	currentAllowance, err := CheckAllowance(ctx, client, tokenAddr, s.Address(), spenderAddr)
	if err != nil {
		logger.Warnf("Could not check allowance: %v", err)
	} else if currentAllowance.Cmp(requiredAmount) >= 0 {
		logger.Debugf("Current allowance (%s) is sufficient for required (%s)", currentAllowance.String(), requiredAmount.String())
		return "", nil
	}

	// Infinite approval: MaxUint256
	maxUint := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))

	var data []byte
	data = append(data, approveMethodID...)
	data = append(data, common.LeftPadBytes(spenderAddr.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(maxUint.Bytes(), 32)...)

	signedTx, err := s.BuildAndSignTx(ctx, client, tokenAddr, big.NewInt(0), data, nonce, 100000)
	if err != nil {
		return "", fmt.Errorf("failed to sign approval transaction: %w", err)
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("failed to broadcast approval transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	logger.Infof("Token approval broadcasted. Tx: %s", txHash)
	return txHash, nil
}

func CheckTokenBalance(
	ctx context.Context,
	client *ethclient.Client,
	tokenAddr common.Address,
	ownerAddr common.Address,
) (*big.Int, error) {
	if tokenAddr == (common.Address{}) {
		return big.NewInt(0), nil
	}

	var data []byte
	data = append(data, balanceOfMethodID...)
	data = append(data, common.LeftPadBytes(ownerAddr.Bytes(), 32)...)

	msg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}

	res, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %w", err)
	}

	if len(res) < 32 {
		return big.NewInt(0), nil
	}

	return new(big.Int).SetBytes(res[:32]), nil
}
