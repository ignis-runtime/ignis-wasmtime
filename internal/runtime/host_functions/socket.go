//go:build !wasip1

package host_functions

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/bytecodealliance/wasmtime-go/v41"
)

// DefineLegacyWasiSockets adds legacy WASI preview 1 socket functions to the linker
func DefineLegacyWasiSockets(linker *wasmtime.Linker) error {
	// Define the sock_setsockopt function that was requested
	// This is a stub implementation that returns success (0) for now
	// In a real implementation, this would need to actually set socket options
	sockSetSockOptFunc := func(caller *wasmtime.Caller, fd, level, optname, optval, optlen int32) int32 {
		// For now, return 0 (success) to allow the WASM module to continue
		// In a production implementation, this would need to actually handle socket options
		return 0
	}

	// Add the function to the linker under the expected name
	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_setsockopt", sockSetSockOptFunc); err != nil {
		return fmt.Errorf("failed to define sock_setsockopt: %w", err)
	}

	// Add other common socket functions that might be expected by legacy WASI modules
	sockGetSockOptFunc := func(caller *wasmtime.Caller, fd, level, optname, optval, optlen int32) int32 {
		// For now, return 0 (success) to allow the WASM module to continue
		return 0
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_getsockopt", sockGetSockOptFunc); err != nil {
		return fmt.Errorf("failed to define sock_getsockopt: %w", err)
	}

	sockBindFunc := func(caller *wasmtime.Caller, fd, addr, addrlen int32) int32 {
		// For now, return 0 (success) to allow the WASM module to continue
		return 0
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_bind", sockBindFunc); err != nil {
		return fmt.Errorf("failed to define sock_bind: %w", err)
	}

	sockConnectFunc := func(caller *wasmtime.Caller, fd, addr, addrlen int32) int32 {
		// For now, return 0 (success) to allow the WASM module to continue
		return 0
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_connect", sockConnectFunc); err != nil {
		return fmt.Errorf("failed to define sock_connect: %w", err)
	}

	// Add the sock_getaddrinfo function that was requested in the error
	// This is a stub implementation that writes a null pointer to the result location
	// and returns EAI_NONAME to indicate that the hostname could not be resolved
	sockGetAddrInfoFunc := func(caller *wasmtime.Caller, node, service, hints, addrinfo, maxaddrs, naddrs, flags, family int32) int32 {
		// Get the memory to write the result
		memory := caller.GetExport("memory").Memory()
		if memory == nil {
			return 1 // EAI_BADFLAGS error code
		}

		// Write null pointer (0) to the result address to indicate no results
		// This follows the expected behavior of getaddrinfo when no addresses are found
		data := memory.UnsafeData(caller)
		if addrinfo >= 0 && int(addrinfo)+4 <= len(data) {
			// Write 0 to indicate no results (NULL pointer)
			data[addrinfo] = 0
			data[addrinfo+1] = 0
			data[addrinfo+2] = 0
			data[addrinfo+3] = 0
		}

		// Return EAI_NONAME (8) to indicate that the hostname could not be resolved
		// This allows the program to handle the error gracefully rather than crash
		return 8
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_getaddrinfo", sockGetAddrInfoFunc); err != nil {
		return fmt.Errorf("failed to define sock_getaddrinfo: %w", err)
	}

	// Add other common WASI networking functions that might be expected
	sockOpenFunc := func(caller *wasmtime.Caller, domain, socketType, protocol int32) int32 {
		// For now, return an error to indicate the operation is not supported
		return 58 // ENOTSOCK error code
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_open", sockOpenFunc); err != nil {
		return fmt.Errorf("failed to define sock_open: %w", err)
	}

	// Add sock_listen function that might also be expected
	sockListenFunc := func(caller *wasmtime.Caller, fd, backlog int32) int32 {
		// For now, return an error to indicate the operation is not supported
		return 58 // ENOTSOCK error code
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_listen", sockListenFunc); err != nil {
		return fmt.Errorf("failed to define sock_listen: %w", err)
	}

	// Add sock_getpeeraddr function that might also be expected
	sockGetPeerAddrFunc := func(caller *wasmtime.Caller, fd, addr, addrlen, flags int32) int32 {
		// For now, return an error to indicate the operation is not supported
		return 58 // ENOTSOCK error code
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_getpeeraddr", sockGetPeerAddrFunc); err != nil {
		return fmt.Errorf("failed to define sock_getpeeraddr: %w", err)
	}

	// Add sock_getlocaladdr function that might also be expected
	sockGetLocalAddrFunc := func(caller *wasmtime.Caller, fd, addr, addrlen, flags int32) int32 {
		// For now, return an error to indicate the operation is not supported
		return 58 // ENOTSOCK error code
	}

	if err := linker.FuncWrap("wasi_snapshot_preview1", "sock_getlocaladdr", sockGetLocalAddrFunc); err != nil {
		return fmt.Errorf("failed to define sock_getlocaladdr: %w", err)
	}

	return nil
}

// HostSocketRequest represents the structure received from the guest for socket operations.
type HostSocketRequest struct {
	Operation string `json:"operation"` // "dial", "read", "write", "close"
	Address   string `json:"address"`   // host:port for dial
	FD        int    `json:"fd"`        // file descriptor for read/write/close
	Data      []byte `json:"data"`      // data to write
	Size      int    `json:"size"`      // size to read
	Network   string `json:"network"`   // network type (tcp, udp, etc.)
}

// HostSocketResponse represents the structure sent from the host back to the guest.
type HostSocketResponse struct {
	Error     string `json:"error,omitempty"`
	FD        int    `json:"fd,omitempty"`         // for dial operations
	Data      []byte `json:"data,omitempty"`       // for read operation
	BytesRead int    `json:"bytes_read,omitempty"` // for read operation
	BytesSent int    `json:"bytes_sent,omitempty"` // for write operation
}

// Connection interface
type Connection interface {
	Read(buffer []byte) (int, error)
	Write(data []byte) (int, error)
	Close() error
}

// RealConnection wraps a real network connection
type RealConnection struct {
	conn net.Conn
}

func (rc *RealConnection) Read(buffer []byte) (int, error) {
	return rc.conn.Read(buffer)
}

func (rc *RealConnection) Write(data []byte) (int, error) {
	return rc.conn.Write(data)
}

func (rc *RealConnection) Close() error {
	return rc.conn.Close()
}

// ConnectionStore holds all connections
type ConnectionStore struct {
	realConns map[int]Connection
	nextFD    int
}

var connStore = &ConnectionStore{
	realConns: make(map[int]Connection),
	nextFD:    100,
}

// LinkSocketFunctions attaches the socket host functions to the Wasmtime linker.
func LinkSocketFunctions(store *wasmtime.Store, linker *wasmtime.Linker) error {
	return linker.DefineFunc(store, "env", "host_socket_operation", func(caller *wasmtime.Caller, reqPtr, reqLen, respPtr, respLen int32) int32 {
		memory := caller.GetExport("memory").Memory()
		if memory == nil {
			log.Println("host_socket_operation: failed to get memory export")
			return 0
		}

		// Read the request JSON from guest memory.
		reqBytes := memory.UnsafeData(store)[reqPtr : reqPtr+reqLen]

		// Unmarshal the request.
		var hostReq HostSocketRequest
		if err := json.Unmarshal(reqBytes, &hostReq); err != nil {
			log.Printf("host_socket_operation: failed to unmarshal request JSON: %v\n", err)
			return 0
		}

		// Perform the requested socket operation
		var hostResp HostSocketResponse

		switch hostReq.Operation {
		case "dial":
			conn, err := net.Dial("tcp", hostReq.Address)
			if err != nil {
				hostResp.Error = err.Error()
			} else {
				fd := connStore.nextFD
				connStore.nextFD++
				connStore.realConns[fd] = &RealConnection{conn: conn}
				hostResp.FD = fd
			}
		case "read":
			conn, exists := connStore.realConns[hostReq.FD]
			if !exists {
				hostResp.Error = "invalid file descriptor"
			} else {
				buffer := make([]byte, hostReq.Size)
				n, err := conn.Read(buffer)
				if err != nil {
					hostResp.Error = err.Error()
				} else {
					hostResp.Data = buffer[:n]
					hostResp.BytesRead = n
				}
			}
		case "write":
			conn, exists := connStore.realConns[hostReq.FD]
			if !exists {
				hostResp.Error = "invalid file descriptor"
			} else {
				n, err := conn.Write(hostReq.Data)
				if err != nil {
					hostResp.Error = err.Error()
				} else {
					hostResp.BytesSent = n
				}
			}
		case "close":
			if conn, exists := connStore.realConns[hostReq.FD]; exists {
				delete(connStore.realConns, hostReq.FD)
				conn.Close()
			} else {
				hostResp.Error = "invalid file descriptor"
			}
		default:
			hostResp.Error = "unknown operation"
		}

		// Marshal the response to JSON.
		respBytes, err := json.Marshal(hostResp)
		if err != nil {
			log.Printf("host_socket_operation: failed to marshal response JSON: %v\n", err)
			return 0
		}

		// Check if the result buffer is large enough.
		if int32(len(respBytes)) > respLen {
			log.Printf("host_socket_operation: result buffer too small. Needed %d, have %d\n", len(respBytes), respLen)
			return 0
		}

		// Write the JSON response back into the guest's memory buffer.
		copy(memory.UnsafeData(store)[respPtr:respPtr+int32(len(respBytes))], respBytes)

		// Return the length of the JSON response written.
		return int32(len(respBytes))
	})
}
