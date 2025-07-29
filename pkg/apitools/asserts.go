package apitools

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AssertContentTypeJson is a shortcut for enforcing that an incoming payload
// is JSON
func AssertContentTypeJson(ctx *gin.Context) bool {
	if ctx.ContentType() != "application/json" {
		ctx.AbortWithStatusJSON(http.StatusNotAcceptable, &gin.H{
			"message": fmt.Sprintf("%s: %s", MsgFailedToBind, "invalid content-type"),
		})
		return false
	}

	return true
}
