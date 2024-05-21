// model/transaction.go
package model

import (
    "math/big"
)

type Transaction struct {
    UUID      string   `json:"uuid"`
    RawTx     string   `json:"rawTx"`
    From      string   `json:"from"`
    To        string   `json:"to"`
    Value     *big.Int `json:"value"`
    GasLimit  uint64   `json:"gasLimit"`
    GasPrice  *big.Int `json:"gasPrice"`
    Data      []byte   `json:"data"`
    Nonce     uint64   `json:"nonce"`
}

type CancellationMessage struct {
    UUID string `json:"uuid"`
}