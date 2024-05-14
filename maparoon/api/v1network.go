package api

import (
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

func (v1n *v1NetworkEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/networks", v1n.listNetworks)
	router.GET("/network", v1n.getNetwork)
	router.POST("/network", v1n.createNetwork)
	router.DELETE("/network/:id", v1n.deleteNetwork)
}

func (v1n *v1NetworkEndpoints) listNetworks(ctx *gin.Context) {
	networks, err := v1n.Querier.ListNetworks(ctx)
	if err != nil {
		logrus.Errorf("could not list networks: %#v", err)
		ctx.JSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"networks": networks,
	})
}

func (v1n *v1NetworkEndpoints) getNetwork(ctx *gin.Context) {
	var network database.Network
	if networkIdStr := ctx.Query("id"); networkIdStr != "" {
		networkId, err := strconv.ParseInt(networkIdStr, 10, 0)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, &gin.H{
				"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "id"),
				"error":   err.Error(),
			})
			return
		}

		network, err = v1n.Querier.GetNetworkById(ctx, networkId)
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
			ctx.JSON(http.StatusBadRequest, &gin.H{
				"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "address"),
			})
			return
		}

		network, err := v1n.Querier.GetNetworkByAddress(ctx, []byte(addr))
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

func (v1n *v1NetworkEndpoints) createNetwork(ctx *gin.Context) {
	if ctx.ContentType() != "application/json" {
		ctx.JSON(http.StatusNotAcceptable, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrFailedToBind, "invalid content-type"),
		})
		return
	}

	networkParams := database.CreateNetworkParams{}
	if err := ctx.BindJSON(&networkParams); err != nil {
		logrus.Errorf("failed to bind network details to database.CreateNetworkParams: %s", err)
		ctx.JSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	network, err := v1n.Querier.CreateNetwork(ctx, networkParams)
	if err != nil {
		logrus.Errorf("failed to create network in database: %s", err)
		ctx.JSON(http.StatusInternalServerError, &gin.H{
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

func (v1n *v1NetworkEndpoints) deleteNetwork(ctx *gin.Context) {
	ctx.String(http.StatusOK, "not implemented")
}
