# 🔥 Ignis Runtime

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.23.0-00ADD8?style=for-the-badge&logo=go)
![Wasmtime Version](https://img.shields.io/badge/Wasmtime-v41-6F42C1?style=for-the-badge&logo=webassembly)
![Gin Version](https://img.shields.io/badge/Gin-v1.11.0-00ADD8?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)

A high-performance server that leverages WebAssembly to execute sandboxed modules written in various languages, including Go and JavaScript.

[Features](#-features) • [Installation](#-installation) • [Usage](#-usage) • [Architecture](#-architecture) • [Development](#-development) • [Contributing](#-contributing)

</div>

---

## 🌟 Features

### Core Capabilities
- **🌐 Multi-language Support**: Execute modules written in any language that compiles to WebAssembly (e.g., Go, Rust, C++) and JavaScript.
- **🔒 Sandboxed Execution**: Modules are executed in a secure sandbox using Wasmtime, providing strong isolation from the host system.
- **⚡ High Performance**: Built with Go and Wasmtime, Ignis Runtime is designed for high performance and low latency.
- **🔌 Host Functions**: Modules can interact with the host system through a set of predefined host functions, allowing them to make HTTP requests, open sockets, and more.
- **💾 Module Caching**: Compiled WebAssembly modules are cached in Redis to reduce startup times and improve performance.
- **🛣️ Dynamic Routing**: Requests are routed to the appropriate module based on a UUID in the URL.

### Technical Highlights
- **Go-powered**: Built with Go for high performance and reliability.
- **Wasmtime Execution**: WebAssembly execution powered by [Wasmtime](https://wasmtime.dev/).
- **QuickJS inside Wasm**: JavaScript execution via a [QuickJS](https://bellard.org/quickjs/) module compiled to Wasm.
- **Gin Framework**: HTTP server built with the [Gin](https://gin-gonic.com/) framework.
- **Redis Caching**: Caching backed by [Redis](https://redis.io/).
- **Protobuf Communication**: Data exchange with modules using [Protocol Buffers](https://developers.google.com/protocol-buffers).

---

## 📋 Prerequisites

-   Go 1.23.0 or higher
-   Docker and Docker Compose (for Redis)
-   `tinygo` for building the Go examples

---

## 🚀 Installation

### Using Docker (Recommended for Redis)

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/ignis-runtime/ignis-wasmtime.git
    cd ignis-wasmtime
    ```

2.  **Start Redis:**
    ```bash
    docker run -d --name redis -p 6379:6379 redis
    ```

3.  **Create a `.env` file:**
    Copy the example environment file:
    ```bash
    cp env_example .env
    ```
    The default `REDIS_ADDR` should work with the Docker command above.

### Manual Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/ignis-runtime/ignis-wasmtime.git
    cd ignis-wasmtime
    ```

2.  **Build the examples:**
    To build the Go example, you need to have `tinygo` installed.
    ```bash
    make build-examples
    ```

3.  **Run the server:**
    ```bash
    go run main.go
    ```
    The server will start on port `8080`.

---

## 🔧 Configuration

### Environment Variables

| Variable     | Description               | Required | Default          |
|--------------|---------------------------|----------|------------------|
| `REDIS_ADDR` | Address of the Redis server | Yes      | `localhost:6379` |

---

## 📱 Usage

You can execute a module by sending an HTTP request to `http://localhost:8080/{uuid}/{path}`, where:

-   `{uuid}` is the UUID of the runtime configuration.
-   `{path}` is the path that will be passed to the module.

**Example: Running the Go Module**
```bash
curl http://localhost:8080/cdda4d36-8943-4033-9caa-e60f89574061/
```

**Example: Running the JavaScript Module**
```bash
curl http://localhost:8080/cdda4d36-8943-4033-9caa-e60f89574060/
```

---

## 🏗️ Architecture

Ignis Runtime employs a robust, modular architecture designed to efficiently execute sandboxed WebAssembly and JavaScript modules. The system orchestrates multiple components to handle incoming HTTP requests, manage module lifecycles, and provide host-level functionalities.

<div align="center">

```
+-------------------+
|   HTTP Request    |
| (Gin Web Server)  |
+-------------------+
        |
        v
+-------------------+
|  Server Handler   |
+-------------------+
        |
        v
+-------------------+
| Runtime Manager   |
| (Selects Runtime) |
+-------------------+
        |
+-------+----------+-------+
|                          |
v                          v
+------------------+       +-------------------+
|  Wasm Runtime    |       |   JS Runtime      |
| (Wasmtime)       |       | (QuickJS on Wasm) |
+------------------+       +-------------------+
        |                          |
        +--------------------------+
                     |
                     v
+------------------------------------+
|           Host Functions           |
|          (HTTP, Sockets)           |
+------------------------------------+
                     |
                     v
+------------------------------------+
|            Redis Cache             |
|         (Compiled Modules)         |
+------------------------------------+
```
</div>

### Detailed Component Breakdown

#### 1. 🌐 Gin Web Server (`main.go`, `internal/server/server.go`)
-   **Role:** The entry point for all external communication. It receives incoming HTTP requests, acting as the primary interface for clients.
-   **Details:** Built upon the high-performance [Gin web framework](https://gin-gonic.com/), it provides fast routing and middleware capabilities. It's configured in `main.go` to listen on a specified port (e.g., `8080`), making the server accessible over HTTP.

#### 2. 🚦 Server Handler (`internal/server/server.go`)
-   **Role:** Responsible for parsing incoming HTTP requests and extracting crucial routing information.
-   **Details:** The `HandleWasmRequest` function identifies the `runtimeID` (a UUID) from the URL path (`/:uuid/*path`). This `runtimeID` is critical for determining which specific module configuration should handle the request. It then encapsulates the HTTP request details (method, headers, body, etc.) into a standardized Protocol Buffer message (`types.FDRequest`) for consistent communication with the modules.

#### 3. 🧠 Runtime Manager (Implicit within `internal/server/server.go`)
-   **Role:** Acts as a central dispatcher, responsible for selecting and initializing the correct runtime environment (either WebAssembly or JavaScript) based on the `runtimeID` provided in the request.
-   **Details:** The manager maintains a registry of `runtime.RuntimeConfig` instances. Upon receiving a request, it retrieves the pre-configured `RuntimeConfig` associated with the `runtimeID`. If a compiled module is not yet instantiated or available in the cache, the manager triggers its instantiation, ensuring that the module is ready for execution.

#### 4. 🚀 Runtimes (`internal/runtime/wasm/wasm_runtime.go`, `internal/runtime/js/js_runtime.go`)
-   **Role:** The core execution environments for user-defined code. Ignis supports two primary runtime types, both leveraging Wasmtime for secure and efficient sandboxed execution.
-   **Details:**
    -   **Shared `runtime.Session` (`internal/runtime/runtime.go`):** Both Wasm and JS runtimes are built around the `runtime.Session` concept. A session encapsulates a single, isolated execution context, holding the Wasmtime `Engine`, compiled `Module`, and dedicated I/O descriptors (`stdin`, `stdout` represented as `os.File`s). This isolation prevents interference between concurrent module executions.
    -   **I/O Handling:** For each execution, the incoming `types.FDRequest` (marshaled into bytes) is written to the session's `stdin`. The module's computed output is then captured from its `stdout`, processed, and ultimately returned as the HTTP response.
    -   **WASI Environment:** Wasmtime sessions are rigorously configured with WASI (WebAssembly System Interface). This setup includes redirecting `stdin`/`stdout`, inheriting necessary environment variables, and pre-opening specific directories (e.g., `internal/runtime/js/modules` for the JS runtime) to grant modules controlled access to the host filesystem within the sandbox.

#### 5. 🔌 Host Functions (`internal/runtime/host_functions/host_functions.go`)
-   **Role:** Bridge the gap between the sandboxed WebAssembly environment and the host Go environment, enabling modules to perform privileged operations like network requests or file system interactions.
-   **Details:**
    -   **Definition and Linking:** Host functions (e.g., for HTTP operations and socket management) are implemented directly in Go within the `internal/runtime/host_functions` package. During a session's `runtime.Session.NewStore` initialization, these Go functions are "linked" into the Wasmtime `Store` and `Linker`.
    -   **Namespace Exposure:** Each logical group of host functions is exposed under a specific namespace (e.g., `ignis_http` for HTTP-related functions, `ignis_socket` for socket functions) that WebAssembly modules can import and call.
    -   **Data Exchange:** Communication between Wasm modules and Go host functions primarily occurs via shared memory within the Wasmtime instance. Data structures, such as `types.FDRequest` and `types.FDResponse`, are serialized and deserialized using Protocol Buffers, ensuring efficient, type-safe, and structured data exchange across the Wasm-Go boundary.

#### 6. 💾 Redis Cache (`internal/cache/redis.go`)
-   **Role:** Significantly optimizes performance by storing compiled WebAssembly modules, thereby eliminating redundant and time-consuming compilation steps on subsequent requests.
-   **Details:**
    -   **Cache Key Generation:** A unique cache key (e.g., `js:{runtimeID}` or `wasm:{runtimeID}`) is generated for each distinct module variant, allowing for precise retrieval.
    -   **Serialization:** When a module (be it the embedded `qjs.wasm` for JavaScript or a user-provided Go WebAssembly module) is compiled by Wasmtime's `Engine`, its optimized, compiled binary form is serialized into raw bytes.
    -   **Storage and Retrieval:** This serialized data, crucially coupled with a content hash of the original module, is stored in Redis. Before attempting to compile a module, the system first queries Redis. If a matching key is found and the associated hash confirms the module's integrity, the deserialized module is directly used, leading to substantial performance gains by bypassing the compilation phase.
    -   **Data Structure:** The cached module data is encapsulated within a `types.Module` Protocol Buffer message, which cleanly stores both the serialized module binary and its hash.

### Project Structure
```
ignis-wasmtime/
├── internal/
│   ├── cache/             # 💾 Redis caching logic for compiled modules.
│   ├── config/            # ⚙️ Configuration management for application settings.
│   ├── models/            # 🧱 Core data models and enums (e.g., RuntimeType).
│   ├── runtime/           # 🏃 Central logic for managing Wasmtime sessions and runtimes.
│   │   ├── host_functions/ # 🔌 Go implementations of functions callable from Wasm modules.
│   │   ├── js/            # 📜 JavaScript runtime implementation using QuickJS on Wasm.
│   │   └── wasm/          # 🚀 Generic WebAssembly runtime implementation.
│   └── server/            # 🌐 Gin-based HTTP server setup and request handling.
├── types/                  # 📦 Protocol Buffer definitions for inter-component communication.
├── example/                # 🧪 Contains sample Go and JavaScript WebAssembly modules.
├── Makefile                # 🛠️ Automation scripts for building, testing, and examples.
├── go.mod                  # 📦 Go module definition and dependency management.
└── main.go                 # 🏁 Application entry point and server initialization.
```

---

## 🛠️ Development

### Runtimes

#### WebAssembly Runtime
The WebAssembly runtime executes modules compiled to WASI-compliant WebAssembly. The core logic for the Wasm runtime is in `internal/runtime/wasm/wasm_runtime.go`.

When a request comes in for a Wasm module, the runtime does the following:

1.  **Retrieves the module:** It fetches the Wasm module from the file system.
2.  **Compiles the module:** It compiles the module using `wasmtime.NewModule`.
3.  **Caches the module:** The compiled module is cached in Redis to avoid recompilation.
4.  **Instantiates the module:** It creates a new instance of the module in a sandboxed environment.
5.  **Links Host Functions:** It links the available host functions to the instance.
6.  **Executes the module:** It calls the `_start` function of the module.
7.  **Returns the response:** It returns the data written to `stdout` by the module as the HTTP response.

#### JavaScript Runtime
The JavaScript runtime executes JavaScript code using a WebAssembly-based QuickJS runtime. The core logic is in `internal/runtime/js/js_runtime.go`.

The JS runtime works as follows:

1.  **Loads the QuickJS module:** It loads the `qjs.wasm` module, which is embedded in the server binary.
2.  **Compiles and caches the module:** It compiles the QuickJS Wasm module and caches it in Redis.
3.  **Instantiates the module:** It creates a new instance of the QuickJS module.
4.  **Executes the JavaScript code:** It passes the user's JavaScript code to the QuickJS instance for execution using the `-e` flag.
5.  **Provides a module system:** The runtime pre-opens the `internal/runtime/js/modules` directory, giving the JavaScript code access to a set of built-in modules (e.g., `fs`, `http`, `os`).

### Host Functions
Host functions allow WebAssembly modules to interact with the host system. They are defined in Go and linked to the Wasmtime runtime. The main entry point for linking host functions is `internal/runtime/host_functions/host_functions.go`.

#### How Host Functions are Imported
1.  **Function Definition:** The host functions are defined in Go files within the `internal/runtime/host_functions` directory (e.g., `http.go`, `socket.go`).
2.  **Linking:** The `Link` function in `host_functions.go` calls other functions to link specific sets of host functions.
3.  **Wasmtime Linker:** The `wasmtime.Linker` is used to define the functions in a specific namespace (e.g., `ignis_http`, `ignis_socket`) that the WebAssembly modules can import.

For example, a Go module can import and use an HTTP host function like this:
```go
//go:wasm-module ignis_http
//go:export call_host
func call_host(request unsafe.Pointer, request_len uint32) uint64

// ...

// Make an HTTP request
response_ptr := call_host(ptr, uint32(len(requestBytes)))
```

### Module Caching
To improve performance, Ignis Runtime caches compiled WebAssembly modules in Redis. The caching logic is in `internal/cache/redis.go`.

The caching mechanism works as follows:

1.  **Cache Key:** A unique cache key is generated for each module based on its UUID.
2.  **Serialization:** When a module is compiled for the first time, it is serialized into a byte array.
3.  **Storage:** The serialized module is stored in Redis along with a hash of the module's content.
4.  **Retrieval:** On subsequent requests, the runtime first checks the cache. If the module exists and the hash matches, it is deserialized and used, avoiding the need for recompilation.
5.  **Data Structure:** The cached module is stored as a Protocol Buffer message (`types.Module`), which contains the serialized module data and its hash.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1.  Fork the repository
2.  Create your feature branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

### Code Style
-   Follow standard Go conventions
-   Use `gofmt` for formatting
-   Add tests for new features
-   Update documentation as needed

---

## 📄 License

This project is licensed under the MIT License.
**Note:** There is no `LICENSE` file in the project. Please add one if you want to specify a license.

---

## 🙏 Acknowledgments

-   [Wasmtime](https://github.com/bytecodealliance/wasmtime-go) - WebAssembly runtime
-   [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
-   [Redis](https://github.com/redis/go-redis) - In-memory data store
-   [Protocol Buffers](https://github.com/golang/protobuf) - Data serialization

---

## 📞 Support

For support, please open an issue in the GitHub repository.

---

<div align="center">
Made with ❤️
</div>