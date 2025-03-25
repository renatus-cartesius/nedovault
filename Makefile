.PHONY: proto-gen
proto-gen:
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/api.proto

.PHONY: server-run
server-run:
	@go run cmd/server/main.go

.PHONY: client-run
client-run:
	@go run cmd/client/main.go