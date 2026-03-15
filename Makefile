.PHONY: up down run build test test-docker lint

up:
	docker compose up --build

down:
	docker compose down

run:
	go run cmd/server/main.go

build:
	go build -o bin/ratelimiter cmd/server/main.go

test:
	go test ./... -v

test-docker:
	docker compose run --rm app go test ./... -v

lint:
	go vet ./...
	go build ./...
