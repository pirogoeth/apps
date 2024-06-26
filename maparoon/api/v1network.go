package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/sirupsen/logrus"
)

type v1NetworkEndpoints struct {
	*types.ApiContext
}

func (e *v1NetworkEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/networks", e.listNetworks)
	router.GET("/network", e.getNetwork)
	router.POST("/networks", e.createNetwork)
	router.PUT("/networks/:id", e.updateNetwork)
	router.DELETE("/networks/:id", e.deleteNetwork)
}

func (e *v1NetworkEndpoints) listNetworks(ctx *gin.Context) {
	networks, err := e.Querier.ListNetworks(ctx)
	if err != nil {
		logrus.Errorf("could not list networks: %#v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	if networks == nil {
		networks = []database.Network{}
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"networks": networks,
	})
}

func (e *v1NetworkEndpoints) getNetwork(ctx *gin.Context) {
	var network database.Network
	if networkIdStr := ctx.Query("id"); networkIdStr != "" {
		networkId, err := strconv.ParseInt(networkIdStr, 10, 0)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
				"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "id"),
				"error":   err.Error(),
			})
			return
		}

		network, err = e.Querier.GetNetworkById(ctx, networkId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
				"message": ErrDatabaseLookup,
				"error":   err.Error(),
			})
		}
		ctx.JSON(http.StatusOK, &gin.H{
			"networks": []database.Network{network},
		})
		return
	} else if networkAddrStr := ctx.Query("address"); networkAddrStr != "" {
		var addressStr string
		if strings.Contains(networkAddrStr, "/") {
			split := strings.Split(networkAddrStr, "/")
			addressStr = split[0]
		} else {
			addressStr = networkAddrStr
		}

		addr := net.ParseIP(addressStr)
		if addr == nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
				"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "address"),
			})
			return
		}

		network, err := e.Querier.GetNetworkByAddress(ctx, string(addr))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, &gin.H{
				"message": ErrDatabaseLookup,
				"error":   err.Error(),
			})
		}
		ctx.JSON(http.StatusOK, &gin.H{
			"networks": []database.Network{network},
		})
		return
	}

	ctx.JSON(http.StatusBadRequest, &gin.H{
		"message": ErrNoQueryProvided,
	})
}

func (e *v1NetworkEndpoints) createNetwork(ctx *gin.Context) {
	if ctx.ContentType() != "application/json" {
		ctx.AbortWithStatusJSON(http.StatusNotAcceptable, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrFailedToBind, "invalid content-type"),
		})
		return
	}

	networkParams := database.CreateNetworkParams{
		Comments:   "",
	}
	if err := ctx.BindJSON(&networkParams); err != nil {
		logrus.Errorf("failed to bind network details to database.CreateNetworkParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	_, err := e.Querier.GetNetworkByAddress(ctx, networkParams.Address)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logrus.Errorf("failed to check if network exists: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})

		return
	} else if err == nil {
		ctx.AbortWithStatusJSON(http.StatusConflict, &gin.H{
			"message": fmt.Sprintf("network already exists with address: %s", networkParams.Address),
		})

		return
	}

	network, err := e.Querier.CreateNetwork(ctx, networkParams)
	if err != nil {
		logrus.Errorf("failed to create network in database: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseInsert,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":  "Successfully created network",
		"networks": []database.Network{network},
	})
}

func (e *v1NetworkEndpoints) deleteNetwork(ctx *gin.Context) {
	network, ok := e.getNetworkByPathParam(ctx)
	if !ok {
		return
	}

	err := e.Querier.DeleteNetwork(ctx, network.ID)
	if err != nil {
		logrus.Errorf("failed to delete network: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseDelete,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":  "Network deleted",
		"networks": []*database.Network{network},
	})
}

func (e *v1NetworkEndpoints) updateNetwork(ctx *gin.Context) {
	network, ok := e.getNetworkByPathParam(ctx)
	if !ok {
		return
	}

	networkUpdate := database.UpdateNetworkParams{
		ID:         network.ID,
		Name:       network.Name,
		Comments:   network.Comments,
	}
	if err := ctx.BindJSON(&networkUpdate); err != nil {
		logrus.Warnf("failed to bind network details to database.UpdateNetworkParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	newNetwork, err := e.Querier.UpdateNetwork(ctx, networkUpdate)
	if err != nil {
		logrus.Errorf("failed to update network record: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseUpdate,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":  "Network updated",
		"networks": []database.Network{newNetwork},
	})
}

func (e *v1NetworkEndpoints) getNetworkByPathParam(ctx *gin.Context) (*database.Network, bool) {
	networkIdStr := ctx.Param("id")
	if networkIdStr == "" {
		logrus.Debugf("blank `id` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "id"),
		})
		return nil, false
	}

	networkId, err := strconv.ParseInt(networkIdStr, 10, 0)
	if err != nil {
		logrus.Debugf("invalid `id` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "id"),
			"error":   err.Error(),
		})
		return nil, false
	}

	network, err := e.Querier.GetNetworkById(ctx, networkId)
	if err != nil {
		logrus.Errorf("error fetching network from database (by id): %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return nil, false
	}

	return &network, true
}
