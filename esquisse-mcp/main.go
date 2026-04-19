// Command esquisse-mcp is a lightweight MCP stdio server that exposes
// adversarial_review and gate_review tools for the Esquisse framework.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "esquisse-mcp does not support Windows — run from Linux, macOS, or WSL")
		os.Exit(1)
	}

	projectRoot := flag.String("project-root", "", "project root directory (default: $PWD)")
	flag.Parse()

	if *projectRoot == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot determine working directory: %v", err)
		}
		*projectRoot = pwd
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "esquisse-mcp", Version: "0.1.0"}, nil)
	registerTools(server, *projectRoot)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("server exited: %v", err)
	}
}
