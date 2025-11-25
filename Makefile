.PHONY: fmt
fmt:
#	@gofumpt -l -w .
#	@gofmt -s -w .
#	@gci write --custom-order -s standard -s "prefix(github.com/qtraffics/)" -s "default" .
	@GOOS=linux golangci-lint fmt

.PHONY: fmt_install
fmt_install:
	go install -v mvdan.cc/gofumpt@latest
	go install -v github.com/daixiang0/gci@latest

.PHONY: lint
lint: fmt
	GOOS=linux golangci-lint run

.PHONY: lint_install
lint_install:
	go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

.PHONY: test
test:
	GOOS=linux go test ./...

.PHONY: generate
generate:
	go generate ./...