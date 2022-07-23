lint:
	golangci-lint run

proto:
	protoc --go_out=pkg --go-grpc_out=pkg api/proto/kiddy-line-processor.proto

run-loc:
	go build -o linesProvider.exe cmd/lines-provider/main.go && linesProvider.exe

run-loc2:
	go build -o linesProvider.exe cmd/kiddy-line-processor/main.go && kidyLinesProvider.exe

build:
	docker run -p 8000:8000 mikhailmi/lines-provider

run: build
	docker run -p 8000:8000 mikhailmi/lines-provider