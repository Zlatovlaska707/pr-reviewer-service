# Основные переменные сборки
APP_NAME=pr-reviewer
BIN_DIR=bin
BIN=$(BIN_DIR)/$(APP_NAME)
OAPI?=oapi-codegen
LINTER=golangci-lint
COMPOSE=docker compose
LOAD_CLI=go run ./load/cli

# Объявляем phony-цели, чтобы make не искал одноимённые файлы
.PHONY: build run test lint lint-fix fmt tidy test-clean compose-up compose-down quick-setup full-setup \
	load-test load-test-setup load-test-report load-test-plot generate go-generate install-tools

# cборка бинаря сервиса
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/run

run:
	go run ./cmd/run

test:
	go test ./...

# линтеры и форматирование
# ВАЖНО: Если возникает ошибка "Go language version used to build golangci-lint is lower",
# обновите golangci-lint: make install-tools
lint:
	$(LINTER) run ./...

lint-fix:
	$(LINTER) run ./... --fix

fmt:
	go fmt ./...

# приведение зависимостей к актуальному состоянию
tidy:
	go mod tidy

test-clean:
	go clean -testcache
	go test -p 1 ./...

# инфраструктура под docker compose
compose-up:
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down -v

# быстрый старт окружения (build + up)
quick-setup: compose-up

# полный прогон форматера, линтера и сборки контейнера
full-setup: fmt lint generate compose-up

# цели нагрузочного тестирования
load-test:
	$(LOAD_CLI)

load-test-setup:
	$(LOAD_CLI) -setup-only

load-test-report:
	$(LOAD_CLI) -report

load-test-plot:
	$(LOAD_CLI) -plot

# генерация Go-DTO из OpenAPI
generate:
	$(OAPI) -generate types -package api -o internal/api/types.gen.go openapi.yml

# go:generate команды
go-generate:
	go generate ./...

# установка тулов для разработки
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install golang.org/x/tools/cmd/goimports@latest

