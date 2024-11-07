package apitools

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AssertContentTypeJson(ctx *gin.Context) bool {
	if ctx.ContentType() != "application/json" {
		ctx.AbortWithStatusJSON(http.StatusNotAcceptable, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrFailedToBind, "invalid content-type"),
		})
		return false
	}

	return true
}
