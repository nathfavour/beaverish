package signer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/ethereum/go-ethereum/crypto"
)

type GeneratedWallet struct {
	Mnemonic   string `json:"mnemonic,omitempty"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	Source     string `json:"source"`
}

// GenerateWallet attempts to use hdwallet-cli first; if unavailable, falls back to native secp256k1 key generation.
func GenerateWallet() (*GeneratedWallet, error) {
	// 1. Try hdwallet-cli first
	if path, err := exec.LookPath("hdwallet-cli"); err == nil && path != "" {
		cmd := exec.Command("hdwallet-cli", "-new", "-network", "eth", "-json")
		output, err := cmd.Output()
		if err == nil {
			var parsed struct {
				Mnemonic string `json:"mnemonic"`
				Wallets  struct {
					Eth struct {
						Address    string `json:"address"`
						PrivateKey string `json:"private_key"`
					} `json:"eth"`
				} `json:"wallets"`
			}
			if err := json.Unmarshal(output, &parsed); err == nil && parsed.Wallets.Eth.Address != "" {
				pk := parsed.Wallets.Eth.PrivateKey
				if len(pk) > 0 && pk[:2] != "0x" {
					pk = "0x" + pk
				}
				return &GeneratedWallet{
					Mnemonic:   parsed.Mnemonic,
					Address:    parsed.Wallets.Eth.Address,
					PrivateKey: pk,
					Source:     "hdwallet-cli",
				}, nil
			}
		}
	}

	// 2. Native Go crypto fallback (secp256k1)
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate native ECDSA key: %w", err)
	}

	pubKey := privKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*pubKey).Hex()
	privBytes := crypto.FromECDSA(privKey)
	privHex := "0x" + hex.EncodeToString(privBytes)

	return &GeneratedWallet{
		Address:    address,
		PrivateKey: privHex,
		Source:     "native-crypto",
	}, nil
}
