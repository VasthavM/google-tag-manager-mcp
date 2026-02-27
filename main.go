package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vasthavm/google-tag-manager-mcp/gtm"
)

const (
	serverName    = "google-tag-manager-mcp"
	serverVersion = "1.0.0"
)

func main() {
	// Structured logging to stderr (stdout is reserved for MCP stdio transport)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Obtain OAuth2 token source (browser flow on first run, saved token thereafter)
	tokenSource, err := getTokenSource(ctx)
	if err != nil {
		logger.Error("authentication failed", "error", err)
		os.Exit(1)
	}

	// Create GTM API client
	client, err := gtm.NewClient(ctx, tokenSource)
	if err != nil {
		logger.Error("failed to create GTM client", "error", err)
		os.Exit(1)
	}
	gtm.SetSharedClient(client)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	// Register utility tools
	registerUtilityTools(server)

	// Register all GTM tools, resources, and prompts
	gtm.RegisterTools(server)

	logger.Info("google-tag-manager-mcp started", "version", serverVersion, "transport", "stdio")

	// Run stdio transport (blocks until stdin closes or signal received)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// registerUtilityTools adds ping and auth_status tools.
func registerUtilityTools(server *mcp.Server) {
	type PingInput struct {
		Message string `json:"message,omitempty" jsonschema:"Optional message to echo back"`
	}
	type PingOutput struct {
		Reply     string `json:"reply"`
		Timestamp string `json:"timestamp"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Test connectivity to the GTM MCP server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PingInput) (*mcp.CallToolResult, PingOutput, error) {
		reply := "pong"
		if input.Message != "" {
			reply = fmt.Sprintf("pong: %s", input.Message)
		}
		return nil, PingOutput{Reply: reply, Timestamp: time.Now().UTC().Format(time.RFC3339)}, nil
	})

	type AuthStatusInput struct{}
	type AuthStatusOutput struct {
		Authenticated bool   `json:"authenticated"`
		Message       string `json:"message"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "auth_status",
		Description: "Check authentication status with Google Tag Manager",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AuthStatusInput) (*mcp.CallToolResult, AuthStatusOutput, error) {
		return nil, AuthStatusOutput{
			Authenticated: true,
			Message:       "Authenticated and ready to access GTM data",
		}, nil
	})
}
