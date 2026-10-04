.PHONY: run run-db test cover fmt vet tidy build docker up down

run:            ## Lance l'API avec le stockage en mémoire
	go run ./cmd/api

run-db:         ## Lance l'API branchée sur le Postgres de docker compose
	DATABASE_URL="postgres://tasks:tasks@localhost:5432/tasks?sslmode=disable" go run ./cmd/api

test:           ## Tests + détecteur de data races
	go test -race ./...

cover:
	go test -coverprofile=cover.out ./... && go tool cover -html=cover.out

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

up:             ## API + Postgres dans Docker
	docker compose up --build

down:
	docker compose down -v
