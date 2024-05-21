package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "rpcproxy/config"
    "rpcproxy/eth"
    "rpcproxy/gas"
    "rpcproxy/kafka"
    "rpcproxy/redis"
    "rpcproxy/rpc"
    "rpcproxy/transaction"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

   
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create Kafka producer and consumer
    kafkaProducer := kafka.NewKafkaProducer(cfg.KafkaBroker, cfg.KafkaTopic)
    defer kafkaProducer.Close()

    kafkaConsumer := kafka.NewKafkaConsumer(cfg.KafkaBroker, cfg.KafkaTopic, "transaction-processor")
    defer kafkaConsumer.Close()

    // Create Redis client
    redisClient, err := redis.NewRedisClient(cfg.RedisURL)
    if err != nil {
        log.Fatalf("Failed to create Redis client: %v", err)
    }
    defer redisClient.Close()

    // Create gas price monitor
    gasMonitor, err := gas.NewGasMonitor(ctx, cfg.InfuraURL)
    if err != nil {
        log.Fatalf("Failed to create gas price monitor: %v", err)
    }
    defer gasMonitor.Close()

    // Create Ethereum client
    ethClient, err := eth.NewEthClient(ctx, cfg.InfuraURL)
    if err != nil {
        log.Fatalf("Failed to create Ethereum client: %v", err)
    }
    defer ethClient.Close()

    // Create transaction service
    txService := transaction.NewTransactionService(ethClient, kafkaConsumer, redisClient)

    // Create JSON-RPC server
    rpcServer := rpc.NewServer(kafkaProducer)

    
    var wg sync.WaitGroup
    wg.Add(3)

    // Start gas price monitoring
	go func() {
		defer wg.Done()
		gasMonitor.Start(ctx, 1*time.Second, redisClient)
		if ctx.Err() != nil {
			log.Printf("Gas price monitoring stopped: %v", ctx.Err())
		}
	}()

    // Start transaction processing
    go func() {
        defer wg.Done()
        err := txService.Start(ctx)
        if err != nil {
            log.Printf("Failed to start transaction service: %v", err)
            cancel()
        }
    }()

    // Start JSON-RPC server
    go func() {
        defer wg.Done()
        err := rpcServer.Start(cfg.ServerPort)
        if err != nil {
            log.Printf("Failed to start JSON-RPC server: %v", err)
            cancel()
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down server...")

    cancel()

    shutdownGracefully(&wg, rpcServer)
}

func shutdownGracefully(wg *sync.WaitGroup, rpcServer *rpc.Server) {
    waitTimeout := 5 * time.Second
    if waitTimeout > 0 {
        done := make(chan struct{})
        go func() {
            defer close(done)
            wg.Wait()
        }()

        select {
        case <-done:
            log.Println("All goroutines finished.")
        case <-time.After(waitTimeout):
            log.Println("Timeout waiting for goroutines to finish. Proceeding with shutdown.")
        }
    } else {
        wg.Wait()
        log.Println("All goroutines finished.")
    }

    if err := rpcServer.Shutdown(); err != nil {
        log.Printf("Error shutting down RPC server: %v", err)
    }
    log.Println("Server gracefully stopped.")
}
