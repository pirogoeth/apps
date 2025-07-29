package system

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/middlewares"
)

func DefaultMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		gin.LoggerWithConfig(gin.LoggerConfig{
			Output:    logrus.StandardLogger().Writer(),
			Formatter: logging.GinJsonLogFormatter,
		}),
		gin.Recovery(),
		middlewares.PrettifyResponseJSON,
	}
}

func NewRouter() *gin.Engine {
	router := gin.New()
	RegisterSystemRoutesTo(router.Group("/system"))

	return router
}

func DefaultRouter() *gin.Engine {
	router := NewRouter()
	for _, mw := range DefaultMiddlewares() {
		router.Use(mw)
	}

	return router
}

func DefaultRouterWithTracing(ctx context.Context, tracingCfg config.TracingConfig) (*gin.Engine, error) {
	router := NewRouter()
	router.Use(otelginMiddlewareEnabler(tracingCfg))
	for _, mw := range DefaultMiddlewares() {
		router.Use(mw)
	}

	return router, nil
}

func otelginMiddlewareEnabler(tracingCfg config.TracingConfig) gin.HandlerFunc {
	downstream := otelgin.Middleware("http")
	return func(ctx *gin.Context) {
		if tracingCfg.Enabled {
			downstream(ctx)
			return
		}

		ctx.Next()
	}
}
