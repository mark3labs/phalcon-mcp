package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server represents the Phalcon MCP server
type Server struct {
	mcpServer *mcpserver.MCPServer
	version   string
}

// NewServer creates a new Phalcon MCP server
func NewServer(version string) *Server {
	s := &Server{
		mcpServer: mcpserver.NewMCPServer(
			"Phalcon MCP",
			version,
		),
		version: version,
	}

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

	// Add get-chain-id-by-name tool
	getChainIdTool := mcp.NewTool("get-chain-id-by-name",
		mcp.WithDescription("Get the chain ID for a blockchain by name, chain, or chainSlug"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the blockchain to look up"),
		),
	)

	// Add tool handlers
	s.mcpServer.AddTool(traceTool, s.traceHandler)
	s.mcpServer.AddTool(profileTool, s.profileHandler)
	s.mcpServer.AddTool(addressLabelTool, s.addressLabelHandler)
	s.mcpServer.AddTool(balanceChangeTool, s.balanceChangeHandler)
	s.mcpServer.AddTool(stateChangeTool, s.stateChangeHandler)
	s.mcpServer.AddTool(transactionOverviewTool, s.transactionOverviewHandler)
	s.mcpServer.AddTool(getChainIdTool, s.getChainIdByNameHandler)

	return s
}

// ServeStdio starts the MCP server in stdio mode
func (s *Server) ServeStdio() error {
	return mcpserver.ServeStdio(s.mcpServer)
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

// fetchBlocksecCookies visits the main site to get cookies with retries
func fetchBlocksecCookies(client *http.Client) error {
	maxRetries := 3
	var lastErr error
	mainPageURL := "https://app.blocksec.com/explorer"

	for attempt := 0; attempt <= maxRetries; attempt++ {

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
			lastErr = fmt.Errorf("failed to send request to main site: %v", err)
			continue
		}

		// Check for successful response
		if mainResp.StatusCode != http.StatusOK {
			mainResp.Body.Close()
			lastErr = fmt.Errorf("main site returned non-200 status code: %d", mainResp.StatusCode)
			continue
		}

		mainResp.Body.Close() // We don't need the body
		return nil            // Success
	}

	// If we've tried maxRetries times and still failed, return the last error
	return fmt.Errorf("failed to fetch cookies after %d attempts: %v", maxRetries, lastErr)
}

// callBlocksecAPI makes an API call to the BlockSec API with retries
func callBlocksecAPI(client *http.Client, endpoint string, chainId int, txHash string) ([]byte, error) {
	// Configure retries
	maxRetries := 3
	var lastErr error
	var respBody []byte

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Prepare the API request with cookies
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
			lastErr = fmt.Errorf("failed to send request to BlockSec API: %v", err)
			continue // Try again
		}

		// Read response body
		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %v", err)
			continue // Try again
		}

		// Check if the request was successful
		if resp.StatusCode == http.StatusOK {
			return respBody, nil // Success, return the response
		}

		// If we got here, the request failed with a non-200 status code
		lastErr = fmt.Errorf("BlockSec API returned non-200 status code: %d - %s", resp.StatusCode, string(respBody))
	}

	// If we've tried maxRetries times and still failed, return the last error
	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// formatJSONResponse formats the response as compact JSON
func formatJSONResponse(respBody []byte) (*mcp.CallToolResult, error) {
	// Return the raw JSON without indentation
	return mcp.NewToolResultText(string(respBody)), nil
}

// handleBlocksecRequest handles all BlockSec API requests using shared code
func (s *Server) handleBlocksecRequest(ctx context.Context, request mcp.CallToolRequest, endpoint string) (*mcp.CallToolResult, error) {
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
func (s *Server) traceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleBlocksecRequest(ctx, request, "trace")
}

// profileHandler handles the profile tool requests
func (s *Server) profileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleBlocksecRequest(ctx, request, "profile")
}

// addressLabelHandler handles the address-label tool requests
func (s *Server) addressLabelHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleBlocksecRequest(ctx, request, "address-label")
}

// balanceChangeHandler handles the balance-change tool requests
func (s *Server) balanceChangeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleBlocksecRequest(ctx, request, "balance-change")
}

// stateChangeHandler handles the state-change tool requests
func (s *Server) stateChangeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleBlocksecRequest(ctx, request, "state-change")
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

// ChainData represents the data for a blockchain from chainlist.org
type ChainData struct {
	Name      string `json:"name"`
	Chain     string `json:"chain"`
	ChainSlug string `json:"chainSlug"`
	ChainId   uint64 `json:"chainId"`
}

// transactionOverviewHandler handles transaction-overview requests by calling all other handlers in parallel
func (s *Server) transactionOverviewHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	resultJSON, err := json.Marshal(overviewResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal overview results: %v", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// fetchChainList fetches the chain list from chainlist.org with retries
func fetchChainList() ([]ChainData, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := http.Get("https://chainlist.org/rpcs.json")
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch chain list: %v", err)
			continue
		}

		// Read body and close response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %v", err)
			continue
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("chainlist.org returned non-200 status code: %d", resp.StatusCode)
			continue
		}

		// Try to unmarshal the response
		var chains []ChainData
		if err := json.Unmarshal(body, &chains); err != nil {
			lastErr = fmt.Errorf("failed to unmarshal chain list: %v", err)
			continue
		}

		// Success - return the chains
		return chains, nil
	}

	// If we've tried maxRetries times and still failed, return the last error
	return nil, fmt.Errorf("failed to fetch chain list after %d attempts: %v", maxRetries, lastErr)
}

// findChainByName searches for a chain by name, chain, or chainSlug
func findChainByName(chains []ChainData, searchTerm string) (string, error) {
	searchTerm = strings.ToLower(strings.TrimSpace(searchTerm))
	if searchTerm == "" {
		return "", fmt.Errorf("search term cannot be empty")
	}

	// Track closest matches
	var nameMatches []ChainData
	var containsMatches []ChainData

	// First pass: look for exact matches or contains
	for _, chain := range chains {
		nameLower := strings.ToLower(chain.Name)
		chainLower := strings.ToLower(chain.Chain)
		slugLower := strings.ToLower(chain.ChainSlug)

		// Check for exact matches first (prioritize these)
		if nameLower == searchTerm || chainLower == searchTerm || slugLower == searchTerm {
			nameMatches = append(nameMatches, chain)
		} else if strings.Contains(nameLower, searchTerm) {
			// If not exact, check if name contains the search term
			containsMatches = append(containsMatches, chain)
		}
	}

	// Return first exact match if found
	if len(nameMatches) > 0 {
		return strconv.FormatUint(nameMatches[0].ChainId, 10), nil
	}

	// Return first contains match if found
	if len(containsMatches) > 0 {
		return strconv.FormatUint(containsMatches[0].ChainId, 10), nil
	}

	return "", fmt.Errorf("no chain found matching '%s'", searchTerm)
}

// getChainIdByNameHandler handles requests to get a chain ID by name
func (s *Server) getChainIdByNameHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the chain name parameter
	chainName, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name must be a string")
	}

	// Fetch the chain list
	chains, err := fetchChainList()
	if err != nil {
		return nil, err
	}

	// Find the chain by name
	chainId, err := findChainByName(chains, chainName)
	if err != nil {
		return nil, err
	}

	// Return the chain ID
	return mcp.NewToolResultText(chainId), nil
}