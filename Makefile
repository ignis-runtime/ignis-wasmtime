run: example-go
	@echo "Running Go program..."
	@go run main.go

example-go: proto
	@GOOS=wasip1 GOARCH=wasm go build -o example/go/index-page/example.wasm example/go/index-page/example.go

tcp-logger: proto
	@GOOS=wasip1 GOARCH=wasm go build -o example/go/tcp-example/tcp_logger.wasm example/go/tcp-example/tcp_logger.go

proto:
	@protoc --go_out=. --go_opt=paths=source_relative --proto_path=. types/types.proto

clean:
	rm -rf $(BUILD_DIR)

.PHONY: example-go tcp-logger run proto setup-build build-wasmtime clean
