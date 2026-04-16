SHELL := /bin/bash

.PHONY: backend-test frontend-test frontend-build build run tidy

backend-test:
	go test ./...

frontend-test:
	cd web && npm test

frontend-build:
	cd web && npm run build

build: frontend-build
	go build -o bin/atomhub ./cmd/atomhub

run:
	go run ./cmd/atomhub

tidy:
	go mod tidy
