package api

import (
	"database/sql"
	"errors"
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

func (e *v1HostPortEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/host/:host_address/ports", e.listHostPorts)
	router.GET("/host/:host_address/port", e.getHostPort)
	router.POST("/host/:host_address/ports", e.createHostPort)
	router.PUT("/host/:host_address/ports/:number/:protocol", e.updateHostPort)
	router.DELETE("/host/:host_address/ports/:number/:protocol", e.deleteHostPort)
}

func (e *v1HostPortEndpoints) listHostPorts(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, e.ApiContext, "host_address")
	if !ok {
		return
	}

	hostPorts, err := e.Querier.ListHostPortsByHostAddress(ctx, host.Address)
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

func (e *v1HostPortEndpoints) getHostPort(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, e.ApiContext, "host_address")
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
		query.Protocol = protocol
	} else {
		// Assume TCP
		query.Protocol = "tcp"
	}

	hostPorts, err := e.Querier.GetHostPort(ctx, query)
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

func (e *v1HostPortEndpoints) createHostPort(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	host, ok := extractHostFromPathParam(ctx, e.ApiContext, "host_address")
	if !ok {
		return
	}

	hostPortParams := database.CreateHostPortParams{
		Address:  host.Address,
		Comments: "",
	}
	if err := ctx.BindJSON(&hostPortParams); err != nil {
		logrus.Errorf("failed to bind host port details to database.CreateHostPortParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	_, err := e.Querier.GetHostPort(ctx, database.GetHostPortParams{
		Address:  host.Address,
		Port:     hostPortParams.Port,
		Protocol: hostPortParams.Protocol,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logrus.Errorf("failed to check if host port exists: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})

		return
	} else if err == nil {
		ctx.AbortWithStatusJSON(http.StatusConflict, &gin.H{
			"message": fmt.Sprintf(
				"host port already exists: %s:%d/%s",
				host.Address,
				hostPortParams.Port,
				hostPortParams.Protocol,
			),
		})

		return
	}

	hostPort, err := e.Querier.CreateHostPort(ctx, hostPortParams)
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

func (e *v1HostPortEndpoints) updateHostPort(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	host, ok := extractHostFromPathParam(ctx, e.ApiContext, "host_address")
	if !ok {
		return
	}

	portNumber, ok := e.getPortNumberByPathParam(ctx)
	if !ok {
		return
	}

	protocol, ok := e.getProtocolByPathParam(ctx)
	if !ok {
		return
	}

	// Load HostPort from database
	hostPort, err := e.Querier.GetHostPort(ctx, database.GetHostPortParams{
		Address:  host.Address,
		Port:     portNumber,
		Protocol: protocol,
	})
	if err != nil {
		logrus.Errorf("could not get host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	hostPortUpdate := database.UpdateHostPortParams{
		Address:  hostPort.Address,
		Port:     hostPort.Port,
		Protocol: hostPort.Protocol,
		Comments: hostPort.Comments,
	}
	if err := ctx.BindJSON(&hostPortUpdate); err != nil {
		logrus.Warnf("failed to bind host port details to database.UpdateHostPortParams: %s", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	newHostPort, err := e.Querier.UpdateHostPort(ctx, hostPortUpdate)
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

func (e *v1HostPortEndpoints) deleteHostPort(ctx *gin.Context) {
	host, ok := extractHostFromPathParam(ctx, e.ApiContext, "host_address")
	if !ok {
		return
	}

	portNumber, ok := e.getPortNumberByPathParam(ctx)
	if !ok {
		return
	}

	protocol, ok := e.getProtocolByPathParam(ctx)
	if !ok {
		return
	}

	// Load HostPort from database
	hostPort, err := e.Querier.GetHostPort(ctx, database.GetHostPortParams{
		Address:  host.Address,
		Port:     portNumber,
		Protocol: protocol,
	})
	if err != nil {
		logrus.Errorf("could not get host port: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseLookup,
			"error":   err.Error(),
		})
		return
	}

	err = e.Querier.DeleteHostPort(ctx, database.DeleteHostPortParams{
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

func (e *v1HostPortEndpoints) getPortNumberByPathParam(ctx *gin.Context) (int64, bool) {
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

func (e *v1HostPortEndpoints) getProtocolByPathParam(ctx *gin.Context) (string, bool) {
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
