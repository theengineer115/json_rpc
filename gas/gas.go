// gas/gas.go
package gas

import (
    "context"
    "fmt"
    "log"
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum/ethclient"
    "rpcproxy/redis"
)

type GasMonitor struct {
    client *ethclient.Client
}

func NewGasMonitor(ctx context.Context, infuraURL string) (*GasMonitor, error) {
    client, err := ethclient.DialContext(ctx, infuraURL)
    if err != nil {
        return nil, err
    }

    return &GasMonitor{
        client: client,
    }, nil
}

func (m *GasMonitor) GetCurrentGasPrice(ctx context.Context) (*big.Int, error) {
    gasPrice, err := m.client.SuggestGasPrice(ctx)
    if err != nil {
        return nil, fmt.Errorf("error suggesting gas price: %w", err)
    }

    return gasPrice, nil
}

// Start continuously fetches the current gas price and stores it in Redis
func (m *GasMonitor) Start(ctx context.Context, interval time.Duration, redisClient *redis.RedisClient) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
			log.Println("Gas price monitoring stopped")
            return
        case <-ticker.C:
            gasPrice, err := m.GetCurrentGasPrice(ctx)
            if err != nil {
                log.Printf("Failed to get current gas price: %v", err)
                continue
            }
            err = redisClient.SetGasPrice(ctx, gasPrice)
            if err != nil {
                log.Printf("Failed to set gas price in Redis: %v", err)
            }
        }
    }
}

func (m *GasMonitor) Close() {
    m.client.Close()
}