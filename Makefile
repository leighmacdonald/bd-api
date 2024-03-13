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
	go install github.com/cosmtrek/air@latest
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	go install github.com/golang-migrate/migrate/v4
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.1
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.6
	go install github.com/goreleaser/goreleaser@latest

dev_db:
	docker-compose --file docker-compose-dev.yml up --force-recreate -V postgres

test:
	go test ./...

update:
	go get -u ./...
