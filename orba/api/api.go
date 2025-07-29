package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	mcpServer "github.com/mark3labs/mcp-go/server"

	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/types"
	api "github.com/pirogoeth/apps/pkg/apitools"
)

var (
	ErrDatabaseDelete = "database delete failed"
	ErrDatabaseInsert = "database insert failed"
	ErrDatabaseLookup = "database lookup failed"
	ErrDatabaseUpdate = "database update failed"

	ErrUserLookup = "database `user` lookup failed"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) error {
	groupV1 := router.Group("/v1")

	sseServer := mcpServer.NewSSEServer(
		apiContext.MCPServer,
		mcpServer.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
			pathSplit := strings.Split(r.RequestURI, "/")
			if len(pathSplit) >= 3 && pathSplit[1] == "v1" && pathSplit[2] == "mcp" {
				return fmt.Sprintf("/v1/mcp/%s", pathSplit[3])
			}

			return ""
		}),
		mcpServer.WithUseFullURLForMessageEndpoint(true),
	)
	groupV1.GET("/mcp/:userId/sse", interposingWrapH(apiContext, sseServer.SSEHandler()))
	groupV1.POST("/mcp/:userId/message", interposingWrapH(apiContext, sseServer.MessageHandler()))

	(&v1Memories{apiContext}).RegisterRoutesTo(groupV1)
	(&v1Users{apiContext}).RegisterRoutesTo(groupV1)

	return nil
}

// interposingWrapH (name currently under development...) extracts important parameters from
// the gin context and injects them in a way that can be consumed from the downstream tool functions
func interposingWrapH(apiContext *types.ApiContext, h http.Handler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO(seanj): At some point we will ideally need to add API keys to this, being a network
		// service with possibly sensitive information. Just knowing the user's ID is def not enough.

		// Snag the user before we parlay into net/http-land
		user, err := getUserFromPathParam(ctx, apiContext.Querier)
		if err != nil {
			api.Bail(ctx, api.ErrorPayload("could not get user from MCP request", err))
			return
		}

		// Extract Context from underlying http.Request
		reqContext := ctx.Request.Context()
		// Set the user parameter on a child context
		reqContextNew := context.WithValue(reqContext, "user", user)
		// Clone the request object w/ the new context
		reqNew := ctx.Request.Clone(reqContextNew)
		// Use the cloned request on the call to the handler
		h.ServeHTTP(ctx.Writer, reqNew)
	}
}

func getUserFromPathParam(ctx *gin.Context, querier *database.Queries) (*database.User, error) {
	userId, err := api.GetPathParamInteger(ctx, "userId")
	if err != nil {
		return nil, fmt.Errorf("%s: could not get parameter as integer: %w", api.MsgInvalidParameter, err)
	}

	user, err := querier.GetUserById(ctx.Request.Context(), userId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrUserLookup, err)
	}

	return &user, nil
}
