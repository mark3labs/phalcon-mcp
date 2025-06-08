package client

import (
	"context"
	
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/phalcon-mcp/server"
)

type InProcessClient struct {
	mcpClient *client.Client
	version   string
}

func NewInProcessClient(phalconServer *server.Server) (*InProcessClient, error) {
	mcpClient, err := client.NewInProcessClient(phalconServer.GetMCPServer())
	if err != nil {
		return nil, err
	}

	return &InProcessClient{
		mcpClient: mcpClient,
		version:   phalconServer.GetVersion(),
	}, nil
}

func (c *InProcessClient) Connect(ctx context.Context) error {
	return c.mcpClient.Start(ctx)
}

func (c *InProcessClient) Initialize(ctx context.Context) error {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "todo",
		Version: c.version,
	}

	_, err := c.mcpClient.Initialize(ctx, initRequest)
	return err
}

func (c *InProcessClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	tools, err := c.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	return tools.Tools, nil
}

func (c *InProcessClient) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "test-tool"
	request.Params.Arguments = arguments

	return c.mcpClient.CallTool(ctx, request)
}

func (c *InProcessClient) Close() error {
	return c.mcpClient.Close()
}
