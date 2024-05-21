// redis/redis.go
package redis

import (
    "context"
    "encoding/json"
    "fmt"
    "math/big"

    "github.com/go-redis/redis/v8"
    "rpcproxy/model"
)

type RedisClient struct {
    client *redis.Client
}

func NewRedisClient(redisURL string) (*RedisClient, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)

    _, err = client.Ping(context.Background()).Result()
    if err != nil {
        return nil, err
    }

    return &RedisClient{
        client: client,
    }, nil
}

func (c *RedisClient) SetTransaction(uuid string, tx *model.Transaction) error {
    txBytes, err := json.Marshal(tx)
    if err != nil {
        return fmt.Errorf("error marshaling transaction: %w", err)
    }
    err = c.client.Set(context.Background(), uuid, txBytes, 0).Err()
    if err != nil {
        return fmt.Errorf("error setting transaction: %w", err)
    }
    return nil
}

func (c *RedisClient) GetTransaction(uuid string) (*model.Transaction, error) {
    txBytes, err := c.client.Get(context.Background(), uuid).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, nil
        }
        return nil, err
    }
    var tx model.Transaction
    err = json.Unmarshal(txBytes, &tx)
    if err != nil {
        return nil, err
    }
    return &tx, nil
}

func (c *RedisClient) RemoveTransaction(uuid string) error {
    err := c.client.Del(context.Background(), uuid).Err()
    if err != nil {
        return err
    }
    return nil
}

// SetGasPrice stores the latest gas price in Redis
func (c *RedisClient) SetGasPrice(ctx context.Context, gasPrice *big.Int) error {
    return c.client.Set(ctx, "gas_price", gasPrice.String(), 0).Err()
}

// SubscribeGasPrice subscribes to gas price updates using Redis Pub/Sub
func (c *RedisClient) SubscribeGasPrice(ctx context.Context, channel string) (*redis.PubSub, error) {
    pubsub := c.client.Subscribe(ctx, channel)
    _, err := pubsub.Receive(ctx)
    if err != nil {
        return nil, err
    }
    return pubsub, nil
}

func (c *RedisClient) Close() error {
    return c.client.Close()
}