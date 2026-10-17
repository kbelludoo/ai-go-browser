package main

import (
	"fmt"
	"os"

	"ai-go-browser/pkg/mcp"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("ai-go-browser-mcp v1.0.0")
		return
	}

	server := mcp.NewServer()
	// Run MCP over standard input/output
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
