all: fmt check build

build:
	go build -o bd-api

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .

check: lint_golangci static

lint_golangci:
	@golangci-lint run --timeout 3m ./...

fix: fmt
	@golangci-lint run --fix

static:
	@staticcheck -go 1.22 ./...

check_deps:
	go install github.com/golang-migrate/migrate/v4
	go install github.com/daixiang0/gci@v0.13.1
	go install mvdan.cc/gofumpt@v0.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.7
	go install github.com/goreleaser/goreleaser@v1.24.0

dev_db:
	docker-compose --file docker-compose-dev.yml up --force-recreate -V postgres

test:
	go test ./...

update:
	go get -u ./...
