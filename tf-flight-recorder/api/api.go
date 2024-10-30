package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
)

const (
	ErrDatabaseDelete = "database delete failed"
	ErrDatabaseInsert = "database insert failed"
	ErrDatabaseLookup = "database lookup failed"
	ErrDatabaseUpdate = "database update failed"
	// ErrInvalidParameter is used when the parameter value can't be parsed or is otherwise invalid
	ErrInvalidParameter = "invalid parameter value"
	// ErrNoQueryProvided is used when no query parameter is provided but is expected
	ErrNoQueryProvided = "no query parameter provided"
	// ErrFailedToBind is used when request body failed to bind to the destination object
	ErrFailedToBind = "failed to bind parameters"
)

func MustRegister(router *gin.Engine, apiContext *types.ApiContext) {
	groupV1 := router.Group("/v1")
}

func extractHostFromPathParam(ctx *gin.Context, endpointCtx *types.ApiContext, paramName string) (*database.Host, bool) {
	hostAddress := ctx.Param(paramName)
	if hostAddress == "" {
		logrus.Debugf("blank `%s` parameter provided", paramName)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, paramName),
		})
		return nil, false
	}

	host, err := endpointCtx.Querier.GetHost(ctx, hostAddress)
	if err != nil {
		logrus.Errorf("error fetching host from database (by address): %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return nil, false
	}

	return &host, true
}

func queryOr(ctx *gin.Context, key, defaultValue string) string {
	value := ctx.Query(key)
	if value == "" {
		return defaultValue
	}

	return value
}
