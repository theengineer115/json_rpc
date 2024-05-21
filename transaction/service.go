package transaction

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math/big"
    "sync"
    "strconv"
    "encoding/hex"

    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/rlp"
    "rpcproxy/eth"
    "rpcproxy/kafka"
    "rpcproxy/model"
    "rpcproxy/redis"
)

type TransactionService struct {
    rbt         *RBTree
    ethClient   *eth.EthClient
    kafkaReader *kafka.KafkaConsumer
    redisClient *redis.RedisClient
    mu          sync.Mutex
}

func NewTransactionService(ethClient *eth.EthClient, kafkaReader *kafka.KafkaConsumer, redisClient *redis.RedisClient) *TransactionService {
    return &TransactionService{
        rbt:         NewRBTree(),
        ethClient:   ethClient,
        kafkaReader: kafkaReader,
        redisClient: redisClient,
    }
}

func (s *TransactionService) Start(ctx context.Context) error {
    // Subscribe to gas price updates
    pubsub, err := s.redisClient.SubscribeGasPrice(ctx, "gas_price_updates")
    if err != nil {
        return fmt.Errorf("failed to subscribe to gas price updates: %v", err)
    }
    defer pubsub.Close()

    // Process transactions concurrently
    var wg sync.WaitGroup
    done := make(chan struct{})

    go func() {
        wg.Wait()
        close(done)
    }()

    // Start consuming messages from Kafka
    go func() {
        err := s.kafkaReader.ConsumeMessages(ctx, s.handleMessage)
        if err != nil {
            log.Printf("Failed to consume Kafka messages: %v", err)
        }
    }()

    for {
        select {
        case <-ctx.Done():
            wg.Wait()
            return ctx.Err()
        case message := <-pubsub.Channel():
            gasPrice, err := strconv.ParseInt(message.Payload, 10, 64)
            if err != nil {
                log.Printf("Failed to parse gas price: %v", err)
                continue
            }
            wg.Add(1)
            go func() {
                defer wg.Done()
                err := s.processTransactions(ctx, big.NewInt(gasPrice))
                if err != nil {
                    log.Printf("Failed to process transactions: %v", err)
                }
            }()
        case <-done:
            return nil
        }
    }
}

func (s *TransactionService) handleMessage(data []byte) error {
    var msg struct {
        Type string          `json:"type"`
        Data json.RawMessage `json:"data"`
    }
    err := json.Unmarshal(data, &msg)
    if err != nil {
        return err
    }

    switch msg.Type {
    case "transaction":
        var tx model.Transaction
        err := json.Unmarshal(msg.Data, &tx)
        if err != nil {
            return err
        }
        return s.handleTransaction(&tx)
    case "cancellation":
        var cancellationMsg model.CancellationMessage
        err := json.Unmarshal(msg.Data, &cancellationMsg)
        if err != nil {
            return err
        }
        return s.handleCancellation(cancellationMsg.UUID)
    default:
        return fmt.Errorf("invalid message type: %s", msg.Type)
    }
}

func (s *TransactionService) handleTransaction(tx *model.Transaction) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Decode the raw transaction
    var rawTx types.Transaction
    rawTxBytes, err := hex.DecodeString(tx.RawTx[2:])
    if err != nil {
        return fmt.Errorf("failed to decode raw transaction hex: %v", err)
    }

    err = rlp.DecodeBytes(rawTxBytes, &rawTx)
    if err != nil {
        return fmt.Errorf("failed to decode raw transaction: %v", err)
    }

    // Get the sender address
    signer := types.NewEIP155Signer(s.ethClient.ChainID())
    from, err := types.Sender(signer, &rawTx)
    if err != nil {
        return fmt.Errorf("failed to get transaction sender: %v", err)
    }
    tx.From = from.Hex()

    // Update the transaction fields
    if rawTx.To() != nil {
        tx.To = rawTx.To().Hex()
    }
    tx.Value = rawTx.Value()
    tx.GasLimit = rawTx.Gas()
    tx.GasPrice = rawTx.GasPrice()
    tx.Data = rawTx.Data()
    tx.Nonce = rawTx.Nonce()

    // Insert the transaction into the RBTree
    s.rbt.Insert(tx)

    // Store the transaction in Redis
    err = s.redisClient.SetTransaction(tx.UUID, tx)
    if err != nil {
        return fmt.Errorf("failed to store transaction in Redis: %v", err)
    }

    return nil
}

func (s *TransactionService) handleCancellation(uuid string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Remove the transaction from the RBTree
    tx := s.rbt.Remove(uuid)
    if tx != nil {
        // Remove the transaction from Redis
        err := s.redisClient.RemoveTransaction(tx.UUID)
        if err != nil {
            return fmt.Errorf("failed to remove transaction from Redis: %v", err)
        }
    }

    return nil
}

func (s *TransactionService) processTransactions(ctx context.Context, gasPrice *big.Int) error {
    // Get transactions within the acceptable gas price range
    minGasPrice := new(big.Int).Sub(gasPrice, big.NewInt(1000000000))
    maxGasPrice := new(big.Int).Add(gasPrice, big.NewInt(1000000000))
    transactions := s.rbt.GetTransactionsWithinRange(minGasPrice, maxGasPrice)

    // Process transactions concurrently
    var wg sync.WaitGroup
    for _, tx := range transactions {
        wg.Add(1)
        go func(tx *model.Transaction) {
            defer wg.Done()
            err := s.processTransaction(ctx, tx)
            if err != nil {
                log.Printf("Failed to process transaction: %v", err)
            }
        }(tx)
    }
    wg.Wait()
    return nil
}

func (s *TransactionService) processTransaction(ctx context.Context, tx *model.Transaction) error {
    var rawTx types.Transaction
    rawTxBytes, err := hex.DecodeString(tx.RawTx[2:])
    if err != nil {
        return fmt.Errorf("failed to decode raw transaction hex: %v", err)
    }

    err = rlp.DecodeBytes(rawTxBytes, &rawTx)
    if err != nil {
        return fmt.Errorf("failed to decode raw transaction: %v", err)
    }

    err = s.ethClient.SendTransaction(ctx, &rawTx)
    if err != nil {
        return fmt.Errorf("failed to send transaction: %v", err)
    }

    // Remove the transaction from the RBTree and Redis after successful submission
    s.mu.Lock()
    s.rbt.Remove(tx.UUID)
    err = s.redisClient.RemoveTransaction(tx.UUID)
    s.mu.Unlock()
    if err != nil {
        log.Printf("Failed to remove transaction from Redis: %v", err)
    }

    return nil
}
