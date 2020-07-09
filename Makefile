export BIN = ${PWD}/bin
export GOBIN = $(BIN)

pkgs = $(shell go list ./...)

.PHONY: check
check: lint test

.PHONY: lint
lint: $(BIN)/golangci-lint
	$(BIN)/golangci-lint run

.PHONY: fix
fix: $(BIN)/golangci-lint
	$(BIN)/golangci-lint run --fix

.PHONY: test
test:
	go build cmd/test/test.go
	go test

$(BIN)/golangci-lint:
	curl --retry 5 -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.30.0
