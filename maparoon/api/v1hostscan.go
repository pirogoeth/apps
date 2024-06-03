package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/sirupsen/logrus"
)

type v1HostScanEndpoints struct {
	*types.ApiContext
}

func (e *v1HostScanEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.GET("/hostscan", e.getHostScan)
	router.POST("/hostscans", e.createHostScans)
	router.DELETE("/hostscans/:address", e.deleteHostScan)
}

func (e *v1HostScanEndpoints) getHostScan(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusNotImplemented, &gin.H{
		"message": "not implemented",
	})
}

func (e *v1HostScanEndpoints) createHostScans(ctx *gin.Context) {
	if ok := assertContentTypeJson(ctx); !ok {
		return
	}

	req := types.CreateHostScansRequest{}
	if err := ctx.BindJSON(&req); err != nil {
		logrus.Errorf("failed to bind request to types.CreateHostScansRequest: %s", err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrFailedToBind, err.Error()),
		})
		return
	}

	// Check if the referenced network exists
	network, err := e.Querier.GetNetworkById(ctx, req.NetworkId)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrDatabaseLookup, err.Error()),
		})
		return
	}

	// Check that hostscans were provided
	if len(req.HostScans) == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
			"message": fmt.Sprintf("%s: %s", ErrInvalidParameter, "host_scans"),
		})
		return
	}

	// Each host within the hostscans details MUST belong to the referenced network
	for _, hostScan := range req.HostScans {
		_, err := e.Querier.GetHostWithNetwork(ctx, database.GetHostWithNetworkParams{
			NetworkID: network.ID,
			Address:   hostScan.Address,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, &gin.H{
					"message": fmt.Sprintf("host not found in network: %s", hostScan.Address),
				})
			} else {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
					"message": fmt.Sprintf("%s: %s", ErrDatabaseLookup, err.Error()),
				})
			}
			return
		}
	}

	// Index the hostscans
	logrus.Debugf("check-out searcher handle")
	handle := e.Searcher.Handle()
	defer handle.Close()

	batch := handle.Index().NewBatch()
	for _, hostScan := range req.HostScans {
		batch.Index(hostScan.Address, &types.HostScanDocument{
			Address:     hostScan.Address,
			Network:     network,
			ScanDetails: hostScan.ScanDetails,
		})
	}

	// Commit the batch
	logrus.Debugf("committing batch with %d host scans", len(req.HostScans))
	if err := handle.Index().Batch(batch); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrDatabaseInsert,
			"error":   err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusCreated, &gin.H{
		"message": fmt.Sprintf("successfully indexed hostscans: %d operations", batch.Size()),
	})
}

func (e *v1HostScanEndpoints) deleteHostScan(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusNotImplemented, &gin.H{
		"message": "not implemented",
	})
}
