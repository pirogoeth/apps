package api

import (
	"github.com/gin-gonic/gin"

	"github.com/pirogoeth/apps/nomad-deployer/types"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) {
	groupV1 := router.Group("/v1")
	v1Deploy := &v1DeployEndpoints{apiContext}
	v1Deploy.RegisterRoutesTo(groupV1)
}
