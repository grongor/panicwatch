GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: check
check: lint test

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: fix
fix:
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go build cmd/test/test.go
	go test --timeout 5m --count 1 ./...
	go test --timeout 5m --count 1 --race ./...
	go test --timeout 5m --count 10 ./...
