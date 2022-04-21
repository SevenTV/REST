all: rest linux

BUILDER := "unknown"
VERSION := "unknown"

ifeq ($(origin REST_BUILDER),undefined)
	BUILDER = $(shell git config --get user.name);
else
	BUILDER = ${REST_BUILDER};
endif

ifeq ($(origin REST_VERSION),undefined)
	VERSION = $(shell git rev-parse HEAD);
else
	VERSION = ${REST_VERSION};
endif

linux:
	GOOS=linux GOARCH=amd64 go build -v -ldflags "-X 'main.Version=${VERSION}' -X 'main.Unix=$(shell date +%s)' -X 'main.User=${BUILDER}'" -o bin/rest .

lint:
	staticcheck ./...
	go vet ./...
	golangci-lint run

deps:
	go mod download
	go install honnef.co/go/tools/cmd/staticcheck@generics
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

test:
	go test -count=1 -cover ./...

rest:
# Generate docs
	swag init --dir src/server/v3 -g v3.go -o gen/v3/docs
	swag init --dir src/server/v2 -g v2.go -o gen/v2/docs
