.PHONY: dev-build-up
dev-build-up: build ## build and run service
	docker-compose up --build -d

.PHONY: dev-build-down
dev-build-down: ## docker-compose down
	docker-compose down

.PHONY: gen-mocks
gen-mocks: ## gen mocks for interfaces from pkg/service
	mockery -dir pkg/service -all -output ./pkg/mocks

.PHONY: down
down: ## stops containers
	docker-compose down

.PHONY: up
up: ## pull, runs service and all deps
	docker-compose pull && docker-compose up --build -d

.PHONY: upfast
upfast: ## runs service without updating images
	docker-compose up -d

.PHONY: build
build: ## build auth1 executable
	GOOS="linux" GOARCH=amd64 CGO_ENABLED=0 go build -o ./auth1 main.go

.PHONY: test
test: ## run go test
	go test ./...

.PHONY: test-cover
test-cover: ## run go test with coverage
	go test ./... -coverprofile=coverage.out -covermode=atomic

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help