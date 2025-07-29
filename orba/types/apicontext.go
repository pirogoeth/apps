package types

import (
	mcpServer "github.com/mark3labs/mcp-go/server"

	"github.com/pirogoeth/apps/orba/database"
)

type ApiContext struct {
	// Config is the application configuration
	Config *Config

	// Querier is the database interface
	Querier *database.Queries

	// MCPServer is an MCP server that is integrated into an agent of your choosing
	MCPServer *mcpServer.MCPServer
}
