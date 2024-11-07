package apitools

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func QueryOr(ctx *gin.Context, key, defaultValue string) string {
	value := ctx.Query(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func QueryBool(ctx *gin.Context, queryParam string, defaultValue bool) bool {
	queryValue, err := strconv.ParseBool(QueryOr(ctx, queryParam, strconv.FormatBool(defaultValue)))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, queryParam),
		})
		return defaultValue
	}

	return queryValue
}
