package system

import (
	"github.com/gin-gonic/gin"
)

const (
	StatusOk        = "ok"
	StatusUnhealthy = "unhealthy"
)

// RegisterSystemRoutesTo registers all system routes to the given router group
func RegisterSystemRoutesTo(router *gin.RouterGroup) {
	router.GET("/liveness", GetLiveness)
	router.GET("/readiness", GetReadiness)
}

// GetLiveness returns a simple ping response to indicate the service is alive
func GetLiveness(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": StatusOk,
	})
}

// GetReadiness returns a more informative response to indicate readiness of the
// application and its components
func GetReadiness(c *gin.Context) {
	status, components := RunReadinessChecks(c)

	c.JSON(200, gin.H{
		"status":     status,
		"components": components,
	})
}
