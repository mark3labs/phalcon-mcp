package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server in stdio mode",
	Long:  `Start the Model Context Protocol (MCP) server in stdio mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting MCP server in stdio mode...")

		// Create MCP server
		s := server.NewMCPServer(
			"Phalcon MCP",
			Version,
		)

		// Add trace tool
		traceTool := mcp.NewTool("trace",
			mcp.WithDescription("Trace the different calls of a transaction on a blockchain also provide gas usage metrics."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to trace"),
			),
		)

		// Add profile tool
		profileTool := mcp.NewTool("profile",
			mcp.WithDescription("Profile a transaction on a blockchain. Gives details about the transaction, flow of funds and token information."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to trace"),
			),
		)

		// Add address-label tool
		addressLabelTool := mcp.NewTool("address-label",
			mcp.WithDescription("Get human readable labels for contract addresses like tokens, protocols, and other on-chain entities."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to get address labels for"),
			),
		)

		// Add balance-change tool
		balanceChangeTool := mcp.NewTool("balance-change",
			mcp.WithDescription("Retrieve detailed balance change information for a transaction."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to get balance changes for"),
			),
		)

		// Add state-change tool
		stateChangeTool := mcp.NewTool("state-change",
			mcp.WithDescription("Retrieve detailed information about state changes like storage variables in contracts for a transaction."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to get state changes for"),
			),
		)

		// Add transaction-overview tool
		transactionOverviewTool := mcp.NewTool("transaction-overview",
			mcp.WithDescription("Comprehensive overview of a transaction by aggregating data from all available analysis tools."),
			mcp.WithString("chainId",
				mcp.Required(),
				mcp.Description("ID of the blockchain"),
			),
			mcp.WithString("transactionHash",
				mcp.Required(),
				mcp.Description("Hash of the transaction to analyze"),
			),
		)

		// Add tool handlers
		s.AddTool(traceTool, traceHandler)
		s.AddTool(profileTool, profileHandler)
		s.AddTool(addressLabelTool, addressLabelHandler)
		s.AddTool(balanceChangeTool, balanceChangeHandler)
		s.AddTool(stateChangeTool, stateChangeHandler)
		s.AddTool(transactionOverviewTool, transactionOverviewHandler)

		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	},
}

// BlocksecTraceRequest represents the request payload for BlockSec API
type BlocksecTraceRequest struct {
	ChainID int    `json:"chainID"`
	TxnHash string `json:"txnHash"`
	Blocked bool   `json:"blocked"`
}

// extractRequestParams extracts and validates chainId and transactionHash from the request
func extractRequestParams(request mcp.CallToolRequest) (int, string, error) {
	chainIdStr, ok := request.Params.Arguments["chainId"].(string)
	if !ok {
		return 0, "", fmt.Errorf("chainId must be a string")
	}

	// Convert chainId to integer
	chainId, err := strconv.Atoi(chainIdStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid chainId format: %v", err)
	}

	txHash, ok := request.Params.Arguments["transactionHash"].(string)
	if !ok {
		return 0, "", fmt.Errorf("transactionHash must be a string")
	}

	return chainId, txHash, nil
}

// createHTTPClient creates an HTTP client with a cookie jar and browser-like headers
func createHTTPClient() (*http.Client, error) {
	// Create a cookiejar to store cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	// Create HTTP client with the cookiejar
	client := &http.Client{
		Jar: jar,
		// Don't follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client, nil
}

// fetchBlocksecCookies visits the main site to get cookies
func fetchBlocksecCookies(client *http.Client) error {
	mainPageURL := "https://app.blocksec.com/explorer"
	req, err := http.NewRequest("GET", mainPageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to main page: %v", err)
	}

	// Set browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Get main page to retrieve cookies
	mainResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to main site: %v", err)
	}
	mainResp.Body.Close() // We don't need the body

	return nil
}

// callBlocksecAPI makes an API call to the BlockSec API
func callBlocksecAPI(client *http.Client, endpoint string, chainId int, txHash string) ([]byte, error) {
	// Now make the API request with cookies
	reqBody := BlocksecTraceRequest{
		ChainID: chainId,
		TxnHash: txHash,
		Blocked: false,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create a new request for the API
	apiURL := fmt.Sprintf("https://app.blocksec.com/api/v1/onchain/tx/%s", endpoint)
	apiReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %v", err)
	}

	// Set headers to mimic a browser for the API request
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	apiReq.Header.Set("Accept", "application/json, text/plain, */*")
	apiReq.Header.Set("Origin", "https://app.blocksec.com")
	apiReq.Header.Set("Referer", "https://app.blocksec.com/explorer")

	// Send the API request
	resp, err := client.Do(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to BlockSec API: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BlockSec API returned non-200 status code: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// formatJSONResponse formats the response as prettified JSON
func formatJSONResponse(respBody []byte) (*mcp.CallToolResult, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
		return mcp.NewToolResultText(string(respBody)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// handleBlocksecRequest handles all BlockSec API requests using shared code
func handleBlocksecRequest(ctx context.Context, request mcp.CallToolRequest, endpoint string) (*mcp.CallToolResult, error) {
	// Extract and validate parameters
	chainId, txHash, err := extractRequestParams(request)
	if err != nil {
		return nil, err
	}

	// Create HTTP client
	client, err := createHTTPClient()
	if err != nil {
		return nil, err
	}

	// Fetch cookies
	if err := fetchBlocksecCookies(client); err != nil {
		return nil, err
	}

	// Call the API
	respBody, err := callBlocksecAPI(client, endpoint, chainId, txHash)
	if err != nil {
		return nil, err
	}

	// Format and return the response
	return formatJSONResponse(respBody)
}

// traceHandler handles the trace tool requests
func traceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleBlocksecRequest(ctx, request, "trace")
}

// profileHandler handles the profile tool requests
func profileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleBlocksecRequest(ctx, request, "profile")
}

// addressLabelHandler handles the address-label tool requests
func addressLabelHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleBlocksecRequest(ctx, request, "address-label")
}

// balanceChangeHandler handles the balance-change tool requests
func balanceChangeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleBlocksecRequest(ctx, request, "balance-change")
}

// stateChangeHandler handles the state-change tool requests
func stateChangeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleBlocksecRequest(ctx, request, "state-change")
}

// Result represents a single data source result with success/error status
type Result struct {
	Name    string          `json:"name"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// OverviewResult represents the combined result from all data sources
type OverviewResult struct {
	Results map[string]Result `json:"results"`
}

// transactionOverviewHandler handles transaction-overview requests by calling all other handlers in parallel
func transactionOverviewHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Define the endpoints to query
	endpoints := []struct {
		name     string
		endpoint string
	}{
		{"trace", "trace"},
		{"profile", "profile"},
		{"address_label", "address-label"},
		{"balance_change", "balance-change"},
		{"state_change", "state-change"},
	}

	// Extract and validate parameters once
	chainId, txHash, err := extractRequestParams(request)
	if err != nil {
		return nil, err
	}

	// Create HTTP client
	client, err := createHTTPClient()
	if err != nil {
		return nil, err
	}

	// Fetch cookies once
	if err := fetchBlocksecCookies(client); err != nil {
		return nil, err
	}

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup
	// Create a mutex to protect the results map
	var mu sync.Mutex
	// Create the results map
	overviewResult := OverviewResult{
		Results: make(map[string]Result),
	}

	// Process each endpoint in parallel
	for _, e := range endpoints {
		wg.Add(1)
		// Create a closure to capture the current endpoint
		go func(name, endpoint string) {
			defer wg.Done()

			// Call the API
			respBody, err := callBlocksecAPI(client, endpoint, chainId, txHash)

			// Store the result
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				overviewResult.Results[name] = Result{
					Name:    name,
					Success: false,
					Error:   err.Error(),
				}
				return
			}

			// Store the successful result
			overviewResult.Results[name] = Result{
				Name:    name,
				Success: true,
				Data:    respBody,
			}
		}(e.name, e.endpoint)
	}

	// Wait for all requests to complete
	wg.Wait()

	// Convert the overview result to JSON
	resultJSON, err := json.MarshalIndent(overviewResult, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal overview results: %v", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
