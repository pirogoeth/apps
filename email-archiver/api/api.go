package api

import (
	"github.com/gin-gonic/gin"

	"github.com/pirogoeth/apps/email-archiver/types"
)

const (
	// ErrInvalidParameter is used when the parameter value can't be parsed or is otherwise invalid
	ErrInvalidParameter = "invalid parameter value"
	// ErrNoQueryProvided is used when no query parameter is provided but is expected
	ErrNoQueryProvided = "no query parameter provided"
	// ErrFailedToBind is used when request body failed to bind to the destination object
	ErrFailedToBind = "failed to bind parameters"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) {
	groupV1 := router.Group("/v1")

	v1Search := &v1SearchEndpoints{apiContext}
	v1Search.RegisterRoutesTo(groupV1)
}

func queryOr(ctx *gin.Context, key, defaultValue string) string {
	value := ctx.Query(key)
	if value == "" {
		return defaultValue
	}

	return value
}
