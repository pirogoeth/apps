package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/functional/types"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) error {
	// V1 API group
	groupV1 := router.Group("/v1")
	
	// Register function endpoints
	(&v1Functions{apiContext}).RegisterRoutesTo(groupV1)
	
	// Register invocation endpoints
	(&v1Invocations{apiContext}).RegisterRoutesTo(groupV1)

	return nil
}