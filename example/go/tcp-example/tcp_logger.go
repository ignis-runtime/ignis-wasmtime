//go:build wasip1

package main

import (
	"fmt"
	"os"

	"github.com/ignis-runtime/go-sdk/sdk/net"
)

func main() {
	// Connect to our local TCP server using the net package SDK
	conn, err := net.Dial("tcp", "127.0.0.1:30072")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to 127.0.0.1:30072: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Connected to 127.0.0.1:30072\n")

	// Send a simple message
	message := "Hello from WASM TCP client!\n"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send message: %v\n", err)
		conn.Close()
		return
	}

	// Read response from the connection and log to stderr
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read from connection: %v\n", err)
	} else if n > 0 {
		// Log the received data to stderr
		fmt.Fprintf(os.Stderr, "Received %d bytes from 127.0.0.1:30072\n", n)

		// Print the actual data to stderr
		fmt.Fprintf(os.Stderr, "%s", string(buffer[:n]))
	}

	// Close the connection
	conn.Close()
	fmt.Fprintf(os.Stderr, "Connection to 127.0.0.1:30072 closed\n")
}
