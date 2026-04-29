SERVICE ?= identity

.PHONY: tidy fmt test run-gateway run-identity run-wallet run-trading run-market-data run-hypertrader run-stream-bridge migrate-up migrate-down migrate-status docker-up docker-down

tidy:
	go mod tidy

fmt:
	go fmt ./...

test:
	go test ./...

run-gateway:
	go run ./cmd/gateway

run-identity:
	go run ./cmd/identity

run-wallet:
	go run ./cmd/wallet

run-trading:
	go run ./cmd/trading

run-market-data:
	go run ./cmd/market-data

run-hypertrader:
	go run ./cmd/hypertrader

run-stream-bridge:
	go run ./cmd/stream-bridge

migrate-up:
	go run ./cmd/migrate -service $(SERVICE) -command up

migrate-down:
	go run ./cmd/migrate -service $(SERVICE) -command down

migrate-status:
	go run ./cmd/migrate -service $(SERVICE) -command status

docker-up:
	docker compose up -d

docker-down:
	docker compose down
