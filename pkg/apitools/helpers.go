package apitools

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Body = gin.H

// Ok is a shortcut for returning an HTTP 200 with a generic body
func Ok(ctx *gin.Context, body *Body) {
	ctx.JSON(http.StatusOK, body)
}

// Bail is a shortcut for returning an HTTP 500 with a generic body
func Bail(ctx *gin.Context, body *Body) {
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, body)
}

// GetPathParamString is a shortcut for extracting a path param from a request and returning it as a string
func GetPathParamString(ctx *gin.Context, paramName string) (string, error) {
	param := ctx.Param(paramName)
	if param == "" {
		return "", fmt.Errorf("%s: %s", MsgInvalidParameter, paramName)
	}

	return param, nil
}

// GetPathParamInteger is a shortcut for extracting a path param from a request and returning it as an integer
func GetPathParamInteger(ctx *gin.Context, paramName string) (int64, error) {
	stringVal, err := GetPathParamString(ctx, paramName)
	if err != nil {
		return 0, fmt.Errorf("could not get path parameter `%s` value: %w", paramName, err)
	}

	numericVal, err := strconv.ParseInt(stringVal, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("%s: %s: %w", MsgInvalidParameter, paramName, err)
	}

	return numericVal, nil
}

type ErrorWrappedEndpointFn func(*gin.Context) error

func ErrorWrapEndpoint(endpointFn ErrorWrappedEndpointFn) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := endpointFn(ctx); err != nil {
			Bail(ctx, ErrorPayload("an error occurred", err))
			return
		}
	}
}
