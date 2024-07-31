GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: setup-tools
#? setup-tools: Install dev tools
setup-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.0
	go install github.com/vektra/mockery/v2@v2.43.0
	go install golang.org/x/tools/cmd/goimports@v0.21.0
	go install gotest.tools/gotestsum@v1.11.0

.PHONY: test
#? test: Run the unit and integration tests
test: test-int

.PHONY: test-int
#? test-int: Run the integration tests
test-int:
	GOOS=$(GOOS) GOARCH=$(GOARCH) gotestsum --junitfile=coverage-int.xml --jsonfile=coverage-int.json -- \
 		-tags=integration -coverprofile=coverage-int.txt -covermode atomic -race ./tdsclient/.

.PHONY: fmt
#? fmt: Run gofmt
fmt:
	gofmt -s -l -w dbconnector/ tdsclient/

.PHONY: lint
#? lint: Run golangci-lint
lint:
	golangci-lint run ./...

.PHONY: generate
#? generate: Run go generate
generate:
	go generate ./...
