package api

import (
	"github.com/gin-gonic/gin"

	"github.com/pirogoeth/apps/maparoon/types"
)

const (
	ErrDatabaseInsert = "database insert failed"
	ErrDatabaseLookup = "database lookup failed"
	// ErrInvalidParameter is used when the parameter value can't be parsed or is otherwise invalid
	ErrInvalidParameter = "invalid parameter value"
	// ErrNoQueryProvided is used when no query parameter is provided but is expected
	ErrNoQueryProvided = "no query parameter provided"
	// ErrFailedToBind is used when request body failed to bind to the destination object
	ErrFailedToBind = "failed to bind parameters"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) {
	groupV1 := router.Group("/v1")
	v1Network := &v1NetworkEndpoints{apiContext}
	v1Network.RegisterRoutesTo(groupV1)
}
