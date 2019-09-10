FIRST_GOPATH := $(firstword $(subst :, ,$(shell go env GOPATH)))
STATICKCHECK_PATH := $(FIRST_GOPATH)/bin/staticcheck

pkgs = $(shell go list ./...)

.PHONY: check
check: cs vet staticcheck test

.PHONY: cs
cs:
ifeq ($(shell which goimports),)
	@echo "installing missing tool goimports"
	go get -u golang.org/x/tools/cmd/goimports
endif

	diff=$$(goimports -d . ); test -z "$$diff" || (echo "$$diff" && exit 1)

.PHONY: cs-fix
cs-fix: format

.PHONY: format
format:
	@goimports -w .

.PHONY: vet
vet:
	go vet $(pkgs)

.PHONY: test
test:
	go build cmd/test/test.go
	go test

.PHONY: staticcheck
staticcheck: $(STATICKCHECK_PATH)
	staticcheck $(pkgs)

$(STATICKCHECK_PATH):
	GO111MODULE=off go get -u honnef.co/go/tools/cmd/staticcheck
