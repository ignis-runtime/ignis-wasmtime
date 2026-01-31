run: example-go
	@echo "Running Go program..."
	@go run main.go

example-go: proto
	@GOOS=wasip1 GOARCH=wasm go build -o example/go/index-page/example.wasm example/go/index-page/example.go

tcp-logger: proto
	@GOOS=wasip1 GOARCH=wasm go build -o example/go/tcp-example/tcp_logger.wasm example/go/tcp-example/tcp_logger.go

proto:
	@buf generate

swagger:
	@echo "Generating Swagger documentation..."
	@go run github.com/swaggo/swag/cmd/swag@latest init --parseDependency --parseInternal

clean:
	rm -rf $(BUILD_DIR)

.PHONY: example-go tcp-logger run proto swagger setup-build build-wasmtime clean
