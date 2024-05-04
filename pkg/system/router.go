package system

import (
	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/middlewares"
	"github.com/sirupsen/logrus"
)

func DefaultRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output:    logrus.StandardLogger().Writer(),
		Formatter: logging.GinJsonLogFormatter,
	}))
	router.Use(gin.Recovery())
	router.Use(middlewares.PrettifyResponseJSON)

	RegisterSystemRoutesTo(router.Group("/system"))

	return router
}
