.PHONY: run test proto

run:
	go run ./cmd/server

test:
	cd ../.. && go test ./services/auction-service/...

proto:
	cd ../.. && protoc --go_out=. --go_opt=module=github.com/yayccc/livebid --go-grpc_out=. --go-grpc_opt=module=github.com/yayccc/livebid api/proto/auction/v1/auction.proto
