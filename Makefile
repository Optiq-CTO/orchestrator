.PHONY: proto fmt test run-locally

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	api/proto/*.proto \
	api/proto/external/*/*.proto

fmt:
	go fmt ./...

test:
	go test ./...

run-locally:
	go run cmd/server/main.go
