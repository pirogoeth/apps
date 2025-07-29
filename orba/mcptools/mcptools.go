package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpServer "github.com/mark3labs/mcp-go/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/types"
)

type (
	ToolFunc       func(context.Context, mcp.CallToolRequest, *database.User) (*mcp.CallToolResult, error)
	ToolDescriptor struct {
		name    string
		fn      ToolFunc
		options []mcp.ToolOption
	}
)

func MustRegister(apiContext *types.ApiContext) error {
	tc := newToolsContainer(apiContext)
	for _, tool := range tc.GetTools() {
		apiContext.MCPServer.AddTool(mcp.NewTool(tool.name, tool.options...), toolMiddleware(tool))
	}

	return nil
}

func NewToolDescriptor(name string, fn ToolFunc, options ...mcp.ToolOption) ToolDescriptor {
	return ToolDescriptor{name, fn, options}
}

// toolMiddleware is a wrapper around a tool function that handles a couple things for the
// tool functions:
// 1. Wraps the context into a new tracing span to show the tool call
// 2. Pulls the user model from the request context and injects it into the underlying tool function
func toolMiddleware(tool ToolDescriptor) mcpServer.ToolHandlerFunc {
	tracer := otel.Tracer("mcptools")
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 1. Tracing
		ctx, span := tracer.Start(ctx, fmt.Sprintf("mcptools/%s", tool.name))
		defer span.End()

		// 2. User data extraction
		user, ok := ctx.Value("user").(*database.User)
		if !ok {
			return nil, fmt.Errorf("could not cast user parameter as *database.User")
		}

		if user == nil {
			return nil, fmt.Errorf("could not get user for MCP request")
		}

		span.SetAttributes(attribute.Int64("userId", user.ID))

		return tool.fn(ctx, req, user)
	}
}

func toolResponseFromItem(item any) (*mcp.CallToolResult, error) {
	msg, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("could not json marshal item: %#v", item)
	}

	return mcp.NewToolResultText(string(msg)), nil
}
