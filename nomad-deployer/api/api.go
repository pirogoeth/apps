package api

import (
	"github.com/gin-gonic/gin"

	"github.com/pirogoeth/apps/nomad-deployer/types"
	"github.com/pirogoeth/apps/pkg/system"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) {
	system.RegisterSystemRoutesTo(router.Group("/sys"))

	groupV1 := router.Group("/v1")
	v1Deploy := &v1DeployEndpoints{apiContext}
	v1Deploy.RegisterRoutesTo(groupV1)
}
