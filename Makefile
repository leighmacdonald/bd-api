all: fmt check
	go build

fmt:
	gofmt -s -w .

check: link_golangci lint_vet lint_imports lint_cyclo lint_golint static

link_golangci:
	@golangci-lint run --timeout 3m

lint_vet:
	@go vet -tags ci ./...

lint_imports:
	@test -z $(goimports -e -d . | tee /dev/stderr)

lint_cyclo:
	@gocyclo -over 45 .

lint_golint:
	@golint -set_exit_status $(go list -tags ci ./...)

static:
	@staticcheck -go 1.19 ./...

check_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

test:
	go test ./...

update:
	go get -u ./...
