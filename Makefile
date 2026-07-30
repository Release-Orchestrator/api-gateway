.PHONY: build test lint run docker-build tidy

BINARY_NAME=api-gateway
DOCKER_IMAGE=ghcr.io/release-orchestrator/api-gateway
VERSION?=dev

build:
	go build -o bin/$(BINARY_NAME) ./cmd

test:
	go test -v ./...

lint:
	golangci-lint run ./...

run: build
	USER_SERVICE_URL="http://localhost:8081" ORDER_SERVICE_URL="http://localhost:8082" PAYMENT_SERVICE_URL="http://localhost:8083" ./bin/$(BINARY_NAME)

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

tidy:
	go mod tidy
