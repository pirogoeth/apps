package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/docker-hermes/types"
	api "github.com/pirogoeth/apps/pkg/apitools"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, apiContext *types.ApiContext) error {
	v1 := router.Group("/api/v1")
	{
		// Container endpoints
		v1.GET("/containers", listContainers(apiContext))
		v1.GET("/containers/:host/:id", getContainer(apiContext))
		v1.POST("/containers/query", queryContainers(apiContext))

		// Label endpoints
		v1.GET("/labels", listLabels(apiContext))
		v1.GET("/labels/:key", getLabelValues(apiContext))

		// Host endpoints
		v1.GET("/hosts", listHosts(apiContext))
	}

	logrus.Info("Registered API routes")
	return nil
}

// listContainers returns all active containers
func listContainers(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		containers, err := apiContext.RedisClient.ListContainers(c.Request.Context())
		if err != nil {
			logrus.Errorf("Failed to list containers: %v", err)
			api.ErrorPayload("failed to list containers", err)
			return
		}

		api.Ok(c, &api.Body{
			"containers": containers,
			"count":      len(containers),
		})
	}
}

// getContainer returns a specific container by host and ID
func getContainer(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Param("host")
		id := c.Param("id")

		container, err := apiContext.RedisClient.GetContainer(c.Request.Context(), host, id)
		if err != nil {
			logrus.Errorf("Failed to get container %s on host %s: %v", id, host, err)
			api.ErrorPayload("failed to get container", err)
			return
		}

		if container == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
			return
		}

		api.Ok(c, &api.Body{
			"container": container,
		})
	}
}

// queryContainers queries containers by label filters
func queryContainers(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var query types.ContainerQuery
		if err := c.ShouldBindJSON(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query format"})
			return
		}

		containers, err := apiContext.RedisClient.QueryByLabels(c.Request.Context(), &query)
		if err != nil {
			logrus.Errorf("Failed to query containers: %v", err)
			api.ErrorPayload("failed to query containers", err)
			return
		}

		api.Ok(c, &api.Body{
			"containers": containers,
			"count":      len(containers),
			"query":      query,
		})
	}
}

// listLabels returns all unique label keys and their values
func listLabels(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		labels, err := apiContext.RedisClient.GetLabels(c.Request.Context())
		if err != nil {
			logrus.Errorf("Failed to list labels: %v", err)
			api.ErrorPayload("failed to list labels", err)
			return
		}

		api.Ok(c, &api.Body{
			"labels": labels,
			"count":  len(labels),
		})
	}
}

// getLabelValues returns all values for a specific label key
func getLabelValues(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		values, err := apiContext.RedisClient.GetLabelValues(c.Request.Context(), key)
		if err != nil {
			logrus.Errorf("Failed to get label values for key %s: %v", key, err)
			api.ErrorPayload("failed to get label values", err)
			return
		}

		api.Ok(c, &api.Body{
			"key":    key,
			"values": values,
			"count":  len(values),
		})
	}
}

// listHosts returns all known hosts
func listHosts(apiContext *types.ApiContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		hosts, err := apiContext.RedisClient.GetHosts(c.Request.Context())
		if err != nil {
			logrus.Errorf("Failed to list hosts: %v", err)
			api.ErrorPayload("failed to list hosts", err)
			return
		}

		api.Ok(c, &api.Body{
			"hosts": hosts,
			"count": len(hosts),
		})
	}
}
