package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/email-archiver/types"
)

type v1SearchEndpoints struct {
	*types.ApiContext
}

func (s *v1SearchEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/search", s.search)
}

func (s *v1SearchEndpoints) search(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusNotImplemented, &gin.H{
		"message": "Not implemented",
	})
}
