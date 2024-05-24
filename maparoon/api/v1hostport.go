package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/sirupsen/logrus"
)

type v1HostPortEndpoints struct {
	*types.ApiContext
}

func (v1hp *v1HostPortEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/host/:host_address/ports", v1hp.listHostPorts)
	router.GET("/host/:host_address/port", v1hp.getHostPort)
	router.POST("/host/:host_address/ports", v1hp.createHostPort)
	router.PUT("/host/:host_address/ports/:number/:protocol", v1hp.updateHostPort)
	router.DELETE("/host/:host_address/ports/:number/:protocol", v1hp.deleteHostPort)
}

func (v1hp *v1HostPortEndpoints) listHostPorts(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, v1hp.ApiContext, "host_address")
	if !ok {
		return
	}

	hostPorts, err := v1hp.Querier.ListHostPortsByHostAddress(ctx, host.Address)
	if err != nil {
		logrus.Errorf("could not list host ports: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return

	}

	ctx.JSON(http.StatusOK, &gin.H{
		"host_ports": hostPorts,
	})
}

func (v1hp *v1HostPortEndpoints) getHostPort(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, v1hp.ApiContext, "host_address")
	if !ok {
		return
	}

	portNumberStr := ctx.Query("number")
	if portNumberStr == "" {
		logrus.Debugf("empty `number` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "number"),
		})
		return
	}

	portNumber, err := strconv.ParseInt(portNumberStr, 10, 0)
	if err != nil {
		logrus.Debugf("invalid `number` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "number"),
		})
		return
	}

	query := database.GetHostPortParams{
		Address: host.Address,
		Port:    portNumber,
	}

	protocol := ctx.Query("protocol")
	if protocol != "" {
		query.Protocol.String = protocol
	}

	hostPorts, err := v1hp.Querier.GetHostPort(ctx, query)
	if err != nil {
		logrus.Errorf("could not get host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"host_ports": hostPorts,
	})
}

func (v1hp *v1HostPortEndpoints) createHostPort(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	host, ok := extractHostFromPathParam(ctx, v1hp.ApiContext, "host_address")
	if !ok {
		return
	}

	hostPortParams := database.CreateHostPortParams{
		Address:    host.Address,
		Comments:   "",
		Attributes: "{}",
	}
	if err := ctx.BindJSON(&hostPortParams); err != nil {
		logrus.Errorf("failed to bind host port details to database.CreateHostPortParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	hostPort, err := v1hp.Querier.CreateHostPort(ctx, hostPortParams)
	if err != nil {
		logrus.Errorf("failed to create host port in database: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseInsert,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":    "Successfully created host port",
		"host_ports": []database.HostPort{hostPort},
	})
}

func (v1hp *v1HostPortEndpoints) updateHostPort(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	host, ok := extractHostFromPathParam(ctx, v1hp.ApiContext, "host_address")
	if !ok {
		return
	}

	portNumber, ok := v1hp.getPortNumberByPathParam(ctx)
	if !ok {
		return
	}

	protocol, ok := v1hp.getProtocolByPathParam(ctx)
	if !ok {
		return
	}

	// Load HostPort from database
	hostPorts, err := v1hp.Querier.GetHostPort(ctx, database.GetHostPortParams{
		Address:  host.Address,
		Port:     portNumber,
		Protocol: sql.NullString{String: protocol},
	})
	if err != nil {
		logrus.Errorf("could not get host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	// With number and protocol, we should only have _one_ port
	// Database constraints should prevent this from being more than one
	// If zero, return 404
	if len(hostPorts) == 0 {
		ctx.AbortWithStatusJSON(http.StatusNotFound, &gin.H{
			"message": "Host port not found",
		})
		return
	}

	hostPort := hostPorts[0]
	hostPortUpdate := database.UpdateHostPortParams{
		Address:    hostPort.Address,
		Port:       hostPort.Port,
		Protocol:   hostPort.Protocol,
		Comments:   hostPort.Comments,
		Attributes: hostPort.Attributes,
	}
	if err := ctx.BindJSON(&hostPortUpdate); err != nil {
		logrus.Warnf("failed to bind host port details to database.UpdateHostPortParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	newHostPort, err := v1hp.Querier.UpdateHostPort(ctx, hostPortUpdate)
	if err != nil {
		logrus.Errorf("failed to update host port record: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseUpdate,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":    "Host port updated",
		"host_ports": []database.HostPort{newHostPort},
	})
}

func (v1hp *v1HostPortEndpoints) deleteHostPort(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, v1hp.ApiContext, "host_address")
	if !ok {
		return
	}

	portNumber, ok := v1hp.getPortNumberByPathParam(ctx)
	if !ok {
		return
	}

	protocol, ok := v1hp.getProtocolByPathParam(ctx)
	if !ok {
		return
	}

	// Load HostPort from database
	hostPorts, err := v1hp.Querier.GetHostPort(ctx, database.GetHostPortParams{
		Address:  host.Address,
		Port:     portNumber,
		Protocol: sql.NullString{String: protocol},
	})
	if err != nil {
		logrus.Errorf("could not get host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	// With number and protocol, we should only have _one_ port
	// Database constraints should prevent this from being more than one
	// If zero, return 404
	if len(hostPorts) == 0 {
		ctx.AbortWithStatusJSON(http.StatusNotFound, &gin.H{
			"message": "Host port not found",
		})
		return
	}

	hostPort := hostPorts[0]
	err = v1hp.Querier.DeleteHostPort(ctx, database.DeleteHostPortParams{
		Address:  hostPort.Address,
		Port:     hostPort.Port,
		Protocol: hostPort.Protocol,
	})
	if err != nil {
		logrus.Errorf("could not delete host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseDelete,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, &gin.H{
		"message":    "Host port deleted",
		"host_ports": []database.HostPort{hostPort},
	})
}

//
// parameter helpers
//

func (v1hp *v1HostPortEndpoints) getPortNumberByPathParam(ctx *gin.Context) (int64, bool) {
	portNumberStr := ctx.Param("number")
	if portNumberStr == "" {
		logrus.Debugf("empty `number` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "number"),
		})
		return 0, false
	}

	portNumber, err := strconv.ParseInt(portNumberStr, 10, 0)
	if err != nil {
		logrus.Debugf("invalid `number` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "number"),
		})
		return 0, false
	}

	return portNumber, true
}

func (v1hp *v1HostPortEndpoints) getProtocolByPathParam(ctx *gin.Context) (string, bool) {
	protocol := ctx.Param("protocol")
	if protocol == "" {
		logrus.Debugf("empty `protocol` parameter provided")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "protocol"),
		})
		return "", false
	}

	return protocol, true
}
