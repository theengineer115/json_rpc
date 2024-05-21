// eth/client.go
package eth

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)


type EthClient struct {
    client  *ethclient.Client
    chainID *big.Int
}

// NewEthClient creates a new instance of EthClient.
func NewEthClient(ctx context.Context, infuraURL string) (*EthClient, error) {
    client, err := ethclient.DialContext(ctx, infuraURL)
    if err != nil {
        return nil, err
    }

    chainID, err := client.NetworkID(ctx)
    if err != nil {
        return nil, err
    }

    return &EthClient{
        client:  client,
        chainID: chainID,
    }, nil
}

func (c *EthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.client.SendTransaction(ctx, tx)
}

func (c *EthClient) ChainID() *big.Int {
	return c.chainID
}
// Close closes the EthClient connection.
func (c *EthClient) Close() {
    c.client.Close()
}
