# Ethereum Transaction Proxy

This project implements a proxy for handling Ethereum raw transactions. It monitors the gas price and submits transactions to the Ethereum network when the gas price matches the one set in the transaction. The proxy also supports transaction cancellation requests.

## Features

- Accepts raw Ethereum transactions via JSON-RPC.
- Stores transactions in a Red Black tree and a backup Redis for later submission.
- Monitors gas prices and submits transactions when conditions are met.
- Supports transaction cancellation.
- Integrates with Kafka for message handling.

## Tech Stack

- Go
- Ethereum (go-ethereum)
- Redis
- Kafka
- Docker

## Prerequisites

- Docker
- Docker Compose

## Configuration

The configuration is stored in `config.yaml`. Below is an example:

##yaml
server_port: ":8080"
kafka_broker: "kafka:29092"
kafka_topic: "transactions"
redis_url: "redis://redis:6379"
infura_url: "https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID"

## Starting
docker-compose up --build
