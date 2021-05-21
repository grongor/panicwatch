export BIN = ${PWD}/bin
export GOBIN = $(BIN)

.PHONY: check
check: lint test

.PHONY: lint
lint: $(BIN)/golangci-lint
	$(BIN)/golangci-lint run

.PHONY: fix
fix: $(BIN)/golangci-lint
	$(BIN)/golangci-lint run --fix

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go build cmd/test/test.go
	go test --timeout 5m --count 1 ./...
	go test --timeout 5m --count 1 --race ./...
	go test --timeout 5m --count 10 ./...

.PHONY: clean
clean:
	rm -rf bin

$(BIN)/golangci-lint:
	curl --retry 5 -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh
