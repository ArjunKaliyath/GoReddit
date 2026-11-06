.PHONY: setup build run test

setup:
	go mod download
	npm --prefix web ci

build:
	npm --prefix web run build
	go build -o bin/goreddit ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race ./...
	go vet ./...
	npm --prefix web test
	npm --prefix web run build
