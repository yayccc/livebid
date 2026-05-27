.PHONY: run test proto

run:
	go run ./cmd/server

test:
	cd ../.. && go test ./...

proto:
	cd ../.. && protoc --go_out=. --go_opt=module=github.com/yayccc/livebid --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid api/proto/shop/v1/shop.proto
