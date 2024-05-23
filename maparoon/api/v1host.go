package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/sirupsen/logrus"
)

type v1HostEndpoints struct {
	*types.ApiContext
}

func (v1h *v1HostEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/hosts", v1h.listHosts)
	router.GET("/host", v1h.getHost)
	router.POST("/hosts", v1h.createHost)
	router.PUT("/hosts/:id", v1h.updateHost)
	router.DELETE("/hosts/:id", v1h.deleteHost)
}

func (v1h *v1HostEndpoints) listHosts(ctx *gin.Context) {
	hosts, err := v1h.Querier.ListHosts(ctx)
	if err != nil {
		logrus.Errorf("could not list hosts: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	if hosts == nil {
		hosts = []database.Host{}
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"hosts": hosts,
	})
}

func (v1h *v1HostEndpoints) getHost(ctx *gin.Context) {
	var host database.Host
	if hostIdStr := ctx.Query("id"); hostIdStr != "" {
		hostId, err := strconv.ParseInt(hostIdStr, 10, 0)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
				"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "id"),
				"error":   err.Error(),
			})
			return
		}

		host, err = v1h.Querier.GetHostById(ctx, hostId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
				"message": ErrDatabaseLookup,
				"error":   err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, &gin.H{
			"hosts": []database.Host{host},
		})
		return
	} else if address := ctx.Query("address"); address != "" {
		host, err := v1h.Querier.GetHostByAddress(ctx, address)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
				"message": ErrDatabaseLookup,
				"error":   err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, &gin.H{
			"hosts": []database.Host{host},
		})
		return
	}

	ctx.JSON(http.StatusBadRequest, &gin.H{
		"message": ErrNoQueryProvided,
	})
}

func (v1h *v1HostEndpoints) createHost(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	hostParams := database.CreateHostParams{
		Comments:   "",
		Attributes: "{}",
	}
	if err := ctx.BindJSON(&hostParams); err != nil {
		logrus.Errorf("failed to bind host details to database.CreateHostParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	host, err := v1h.Querier.CreateHost(ctx, hostParams)
	if err != nil {
		logrus.Errorf("failed to create host in database: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseInsert,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message": "Successfully created host",
		"hosts":   []database.Host{host},
	})
}

func (v1h *v1HostEndpoints) updateHost(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	host, ok := extractHostFromPathParam(ctx, v1h.ApiContext, "id")
	if !ok {
		return
	}

	hostUpdate := database.UpdateHostParams{
		ID:         host.ID,
		Comments:   host.Comments,
		Attributes: host.Attributes,
	}
	if err := ctx.BindJSON(&hostUpdate); err != nil {
		logrus.Warnf("failed to bind host details to database.UpdateHostParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	newHost, err := v1h.Querier.UpdateHost(ctx, hostUpdate)
	if err != nil {
		logrus.Errorf("failed to update host record: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseUpdate,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message": "Host updated",
		"hosts":   []database.Host{newHost},
	})
}

func (v1h *v1HostEndpoints) deleteHost(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, v1h.ApiContext, "id")
	if !ok {
		return
	}

	err := v1h.Querier.DeleteHost(ctx, host.ID)
	if err != nil {
		logrus.Errorf("failed to delete host: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseDelete,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message": "Host deleted",
		"hosts":   []*database.Host{host},
	})
}
