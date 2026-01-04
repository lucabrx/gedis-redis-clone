package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lucabrx/gedis/aof"
	"github.com/lucabrx/gedis/resp"
)

func main() {
	portPtr := flag.Int("port", 6379, "Port to listen on")
	aofPtr := flag.String("aof", "database.aof", "Path to AOF file")
	flag.Parse()

	port := *portPtr
	aofPath := *aofPtr

	fmt.Printf("Listening on port %d...\n", port)

	aofFile, err := aof.NewAof(aofPath)
	if err != nil {
		fmt.Println("Failed to create AOF file:", err)
		return
	}

	aofFile.Read(func(value resp.Value) {
		command := strings.ToUpper(value.Array[0].Bulk)
		args := value.Array[1:]
		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command in AOF: ", command)
			return
		}
		handler(args, nil)
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d: %v\n", port, err)
		os.Exit(1)
	}

	StartExpirationJob()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-sigChan:
					return
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}
			go handleConnection(conn, aofFile)
		}
	}()

	<-sigChan
	fmt.Println("\nReceived shutdown signal. Closing resources...")

	listener.Close()
	aofFile.Close()
	fmt.Println("Server stopped cleanly.")
}

func handleConnection(conn net.Conn, aofFile *aof.Aof) {
	defer conn.Close()

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	for {
		value, err := reader.Read()
		if err != nil {
			fmt.Println("Client disconnected or error:", err)
			return
		}

		if value.Type != "array" {
			fmt.Println("Invalid request, expected array")
			continue
		}

		if len(value.Array) == 0 {
			continue
		}

		command := strings.ToUpper(value.Array[0].Bulk)
		args := value.Array[1:]

		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(resp.Value{Type: "error", Str: "ERR unknown command '" + command + "'"})
			continue
		}

		if command == "SET" || command == "DEL" {
			aofFile.Write(value)
		}

		result := handler(args, writer)
		if result.Type == "ignore" {
			continue
		}
		writer.Write(result)
	}
}
