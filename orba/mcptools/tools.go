package mcptools

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/types"
)

type toolsContainer struct {
	apiContext *types.ApiContext
}

func newToolsContainer(apiContext *types.ApiContext) *toolsContainer {
	return &toolsContainer{apiContext}
}

func (tc *toolsContainer) GetTools() []ToolDescriptor {
	return []ToolDescriptor{
		NewToolDescriptor(
			"addMemory",
			tc.addMemoryTool,
			mcp.WithDescription("Add a memory for the current user"),
			mcp.WithString(
				"memory",
				mcp.Required(),
				mcp.Description("The memory to add to the user's database"),
			),
			mcp.WithString(
				"source",
				mcp.Required(),
				mcp.Description("The source of the memory. This MUST be a valid source for the user."),
			),
		),
		NewToolDescriptor(
			"listSources",
			tc.listSourcesTool,
			mcp.WithDescription("List the memory sources associated with the current user"),
		),
		NewToolDescriptor(
			"currentDateTime",
			tc.currentDateTimeTool,
			mcp.WithDescription("Retrieve the current date and time"),
		),
		NewToolDescriptor(
			"searchMemories",
			tc.searchMemoriesTool,
			mcp.WithDescription("Performs full text search over memories in the database"),
			mcp.WithString(
				"query",
				mcp.Required(),
				mcp.Description("Search query to run on the database. Must be a valid SQLite3 FTS5 MUST clause"),
			),
		),
	}
}

func (tc *toolsContainer) testTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logrus.Infof("Test tool has been called!")
	logrus.Infof("Request params: %#v", req.GetArguments())
	logrus.Infof("User from context: %#v", ctx.Value("user"))
	return nil, fmt.Errorf("not implemented :3")
}

func (tc *toolsContainer) currentDateTimeTool(ctx context.Context, req mcp.CallToolRequest, _ *database.User) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf("The current datetime is %s", time.Now().Format(time.UnixDate))), nil
}

func (tc *toolsContainer) addMemoryTool(ctx context.Context, req mcp.CallToolRequest, user *database.User) (*mcp.CallToolResult, error) {
	memory, err := req.RequireString("memory")
	if err != nil {
		return nil, fmt.Errorf("memory parameter not provided")
	}

	source, err := req.RequireString("source")
	if err != nil {
		return nil, fmt.Errorf("source parameter not provided")
	}

	memoryItem, err := tc.apiContext.Querier.CreateMemory(ctx, database.CreateMemoryParams{
		UserID:   user.ID,
		Memory:   memory,
		SourceID: source,
	})
	if err != nil {
		return nil, fmt.Errorf("could not store memory: %w", err)
	}

	return toolResponseFromItem(memoryItem)
}

func (tc *toolsContainer) listSourcesTool(ctx context.Context, req mcp.CallToolRequest, user *database.User) (*mcp.CallToolResult, error) {
	sources, err := tc.apiContext.Querier.ListSourcesForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get sources for user: %w", err)
	}

	return toolResponseFromItem(sources)
}

func (tc *toolsContainer) searchMemoriesTool(ctx context.Context, req mcp.CallToolRequest, user *database.User) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return nil, fmt.Errorf("query parameter not provided")
	}

	logrus.Debugf("SearchMemories(user=%v, query=%s)", user, query)
	results, err := tc.apiContext.Querier.SearchMemories(ctx, user, query)
	if err != nil {
		return nil, fmt.Errorf("could not perform search: %w", err)
	}

	return toolResponseFromItem(results)
}
