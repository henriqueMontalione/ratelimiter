.PHONY: up down run build test

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
