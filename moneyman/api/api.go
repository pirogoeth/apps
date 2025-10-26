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
	// groupV1 := router.Group("/v1")

	return nil
}
