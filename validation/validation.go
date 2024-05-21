package validation

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
)

// validation results for each field
type ValidationResult struct {
	Nonce    bool `json:"nonce"`
	GasPrice bool `json:"gasPrice"`
	GasLimit bool `json:"gasLimit"`
	To       bool `json:"to"`
	Value    bool `json:"value"`
}

// checks if a string is a valid hexadecimal and optionally checks its length
func isValidHexString(value string, length int) bool {
	if len(value) < 2 || value[:2] != "0x" {
		return false
	}
	decoded, err := hex.DecodeString(value[2:])
	return err == nil && (length == 0 || len(decoded) == length)
}

// transaction structure
type Transaction struct {
	Nonce    *big.Int
	GasPrice *big.Int
	GasLimit *big.Int
	To       []byte
	Value    *big.Int
	Data     []byte
	V        *big.Int
	R        *big.Int
	S        *big.Int
}


func ValidateTransaction(rawTx string) (ValidationResult, error) {
	// Validate the overall rawTx format
	if len(rawTx) < 2 || rawTx[:2] != "0x" {
		return ValidationResult{}, errors.New("invalid rawTx format")
	}
	rawTxBytes, err := hex.DecodeString(rawTx[2:])
	if err != nil {
		return ValidationResult{}, errors.New("failed to decode rawTx")
	}

	// RLP decoding
	var tx Transaction
	if err := rlpDecode(rawTxBytes, &tx); err != nil {
		return ValidationResult{}, err
	}

	// Validate nonce
	nonceValid := tx.Nonce != nil && tx.Nonce.Sign() >= 0

	// Validate gas price
	gasPriceValid := tx.GasPrice != nil && tx.GasPrice.Sign() >= 0

	// Validate gas limit
	gasLimitValid := tx.GasLimit != nil && tx.GasLimit.Sign() >= 0

	// Validate to address
	toValid := len(tx.To) == 20

	// Validate value
	valueValid := tx.Value != nil && tx.Value.Sign() >= 0

	return ValidationResult{
		Nonce:    nonceValid,
		GasPrice: gasPriceValid,
		GasLimit: gasLimitValid,
		To:       toValid,
		Value:    valueValid,
	}, nil
}


func rlpDecode(data []byte, v interface{}) error {
	return rlp.DecodeBytes(data, v)
}
