package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/types"
	api "github.com/pirogoeth/apps/pkg/apitools"
)

type v1Memories struct {
	*types.ApiContext
}

func (e *v1Memories) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/user/:userId/memories", api.ErrorWrapEndpoint(e.getMemories))
	router.GET("/user/:userId/memories/_search", api.ErrorWrapEndpoint(e.searchMemories))
	router.POST("/user/:userId/memory/:sourceId", api.ErrorWrapEndpoint(e.createMemory))
	router.DELETE("/user/:userId/memory/:id", api.ErrorWrapEndpoint(e.deleteMemoryById))
}

func (e *v1Memories) getMemories(ctx *gin.Context) error {
	user, err := getUserFromPathParam(ctx, e.Querier)
	if err != nil {
		return fmt.Errorf("could not load user from path parameter: %w", err)
	}

	// NOTE: Passing the gin.Context to the DB methods does NOT correctly pass the trace context.
	// Instead, always pass the request Context.
	memories, err := e.Querier.ListMemories(ctx.Request.Context(), user.ID)
	if err != nil {
		return fmt.Errorf("could not list memories for user: %w", err)
	}

	api.Ok(ctx, &gin.H{"memories": memories})
	return nil
}

func (e *v1Memories) searchMemories(ctx *gin.Context) error {
	_, err := getUserFromPathParam(ctx, e.Querier)
	if err != nil {
		return fmt.Errorf("could not load user from path parameter: %w", err)
	}

	// TODO: What is the search interface for memories going to look like?
	return fmt.Errorf(api.MsgNotImplemented)
}

func (e *v1Memories) createMemory(ctx *gin.Context) error {
	user, err := getUserFromPathParam(ctx, e.Querier)
	if err != nil {
		return fmt.Errorf("could not load user from path parameter: %w", err)
	}

	source, err := api.GetPathParamString(ctx, "sourceId")
	if err != nil {
		return fmt.Errorf("%s: %w", api.MsgInvalidParameter, err)
	}

	var params database.CreateMemoryParams
	if err := ctx.ShouldBind(&params); err != nil {
		return fmt.Errorf("%s: %w", api.MsgInvalidParameter, err)
	}

	params.SourceID = source
	params.UserID = user.ID

	if _, err := e.Querier.CreateMemory(ctx.Request.Context(), params); err != nil {
		return fmt.Errorf("could not create memory: %w", err)
	}

	api.Ok(ctx, &gin.H{"params": params})
	return nil
}

func (e *v1Memories) deleteMemoryById(ctx *gin.Context) error {
	return fmt.Errorf(api.MsgNotImplemented)
}
