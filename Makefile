lint:
	golangci-lint run

proto:
	protoc --go_out=pkg --go-grpc_out=pkg api/proto/kiddy-line-processor.proto

tests:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out

build:
	export GOARCH=amd64
	export GOOS=linux
	export CGO_ENABLED=0
	go build -o bin/linesProvider cmd/lines-provider/main.go
	go build -o bin/kiddyLinesProvider cmd/kiddy-line-processor/main.go
	go build -o bin/client cmd/client/main.go

run: lint
	docker compose build --parallel
	docker compose up -d

stop:
	docker compose down

reload:
	make down && make build

verify: lint build tests
