# Phalcon MCP Server

This MCP server integrates with the [BlockSec](https://blocksec.com) platform to provide blockchain transaction analysis tools via the Model Context Protocol (MCP).

The Model Context Protocol (MCP) is a protocol for AI model integration, allowing AI models to access external tools and data sources.

## Components

### Tools

#### Transaction Analysis

- **Trace**
  - Trace the different calls of a transaction on a blockchain along with gas usage metrics
  - Parameters: `chainId` (required), `transactionHash` (required)

- **Profile**
  - Profile a transaction on a blockchain with details about the transaction, flow of funds and token information
  - Parameters: `chainId` (required), `transactionHash` (required)

- **AddressLabel**
  - Get human readable labels for contract addresses like tokens, protocols, and other on-chain entities
  - Parameters: `chainId` (required), `transactionHash` (required)

- **BalanceChange**
  - Retrieve detailed balance change information for a transaction
  - Parameters: `chainId` (required), `transactionHash` (required)

- **StateChange**
  - Retrieve detailed information about state changes like storage variables in contracts for a transaction
  - Parameters: `chainId` (required), `transactionHash` (required)

- **TransactionOverview** 
  - Comprehensive overview of a transaction by aggregating data from all available analysis tools
  - Parameters: `chainId` (required), `transactionHash` (required)

#### Chain Information

- **GetChainIdByName**
  - Get the chain ID for a blockchain by name, chain, or chainSlug
  - Parameters: `name` (required)

## Getting Started

### Installation

#### Using Go Install

```bash
go install github.com/mark3labs/phalcon-mcp@latest
```

### Usage

Start the MCP server:

```bash
phalcon-mcp serve
```

Check the version:

```bash
phalcon-mcp version
```

### Using as a Package

You can import the server in your Go projects:

#### Stdio Mode

```go
import "github.com/mark3labs/phalcon-mcp/server"

func main() {
    // Create a new server with version
    s := server.NewServer("1.0.0")
    
    // Start the server in stdio mode
    if err := s.ServeStdio(); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

#### In-Process Mode

For in-process usage with the mcp-go client library:

```go

import (
	"context"
	"log"

	"github.com/mark3labs/phalcon-mcp/client"
	"github.com/mark3labs/phalcon-mcp/server"
)

func main() {
	// Create the Phalcon MCP server
	phalconServer := server.NewServer("1.0.0")

	// Create an MCP client using the in-process transport
	mcpClient, err := client.NewInProcessClient(phalconServer)

	// Start the transport
	ctx := context.Background()
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer mcpClient.Close()

	// Initialize the client
	if err = mcpClient.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// List available tools
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	log.Printf("all available tools: %+v", tools)

	// Use the tools...
	result, err := mcpClient.CallTool(ctx, "get-chain-id-by-name", map[string]any{
		"name": "ethereum",
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}
	log.Printf("tool call result %v", result)
}
```

### Usage with Model Context Protocol

To integrate this server with apps that support MCP:

```json
{
  "mcpServers": {
    "phalcon": {
      "command": "phalcon-mcp",
      "args": ["serve"]
    }
  }
}
```

### Docker

#### Running with Docker

You can run the Phalcon MCP server using Docker:

```bash
docker run -i --rm ghcr.io/mark3labs/phalcon-mcp:latest serve
```

#### Docker Configuration with MCP

To integrate the Docker image with apps that support MCP:

```json
{
  "mcpServers": {
    "phalcon": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "ghcr.io/mark3labs/phalcon-mcp:latest",
        "serve"
      ]
    }
  }
}
```

## License

MIT
