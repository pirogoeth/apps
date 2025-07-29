package api

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pirogoeth/apps/pkg/apitools"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

type v1Invocations struct {
	*types.ApiContext
}

func (e *v1Invocations) RegisterRoutesTo(router *gin.RouterGroup) {
	// Function invocation endpoint
	router.POST("/invoke/:function_name", apitools.ErrorWrapEndpoint(e.invokeFunction))
	
	// Invocation management endpoints
	invocations := router.Group("/invocations")
	invocations.GET("", apitools.ErrorWrapEndpoint(e.listInvocations))
	invocations.GET("/:id", apitools.ErrorWrapEndpoint(e.getInvocation))
	
	// Function-specific invocation endpoints
	functions := router.Group("/functions")
	functions.GET("/:id/invocations", apitools.ErrorWrapEndpoint(e.listFunctionInvocations))
	functions.GET("/:id/stats", apitools.ErrorWrapEndpoint(e.getFunctionStats))
}

func (e *v1Invocations) invokeFunction(c *gin.Context) error {
	functionName := c.Param("function_name")
	if functionName == "" {
		return fmt.Errorf("%s: function name is required", apitools.MsgInvalidParameter)
	}

	// Look up function by name
	function, err := e.Querier.GetFunctionByName(c.Request.Context(), functionName)
	if err != nil {
		return fmt.Errorf("function not found: %w", err)
	}

	// Get active deployment
	deployment, err := e.Querier.GetActiveDeploymentByFunction(c.Request.Context(), function.ID)
	if err != nil {
		return fmt.Errorf("no active deployment found for function: %w", err)
	}

	// Create invocation record
	invocationID := uuid.New().String()
	_, err = e.Querier.CreateInvocation(c.Request.Context(), database.CreateInvocationParams{
		ID:           invocationID,
		FunctionID:   function.ID,
		DeploymentID: sql.NullString{String: deployment.ID, Valid: true},
		Status:       string(types.InvocationStatusPending),
	})
	if err != nil {
		return fmt.Errorf("failed to create invocation record: %w", err)
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Convert headers
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Convert query parameters
	queryArgs := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			queryArgs[k] = v[0]
		}
	}

	// Create invocation request
	invReq := &types.InvocationRequest{
		FunctionID: function.ID,
		Body:       body,
		Headers:    headers,
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		QueryArgs:  queryArgs,
	}

	// Get compute provider
	provider, err := e.Compute.Get(deployment.Provider)
	if err != nil {
		return fmt.Errorf("compute provider not available: %w", err)
	}

	// Convert types for compute provider
	computeDep := dbDeploymentToComputeDeployment(deployment)
	computeInvReq := typesInvocationRequestToComputeInvocationRequest(invReq)

	// Execute function
	result, err := provider.Execute(c.Request.Context(), computeDep, computeInvReq)
	if err != nil {
		// Update invocation with error
		e.Querier.UpdateInvocationComplete(c.Request.Context(), database.UpdateInvocationCompleteParams{
			ID:     invocationID,
			Status: string(types.InvocationStatusError),
			Error:  sql.NullString{String: err.Error(), Valid: true},
		})
		return fmt.Errorf("function execution failed: %w", err)
	}

	compResult, ok := result.(*compute.InvocationResult)
	if !ok {
		return fmt.Errorf("invalid invocation result type")
	}

	// Convert back to types
	invResult := computeInvocationResultToTypesInvocationResult(compResult)

	// Update invocation with results
	status := string(types.InvocationStatusSuccess)
	if invResult.StatusCode >= 400 {
		status = string(types.InvocationStatusError)
	}

	e.Querier.UpdateInvocationComplete(c.Request.Context(), database.UpdateInvocationCompleteParams{
		ID:                invocationID,
		Status:            status,
		DurationMs:        sql.NullInt64{Int64: invResult.DurationMS, Valid: true},
		MemoryUsedMb:      sql.NullInt64{Int64: int64(invResult.MemoryUsedMB), Valid: true},
		ResponseSizeBytes: sql.NullInt64{Int64: invResult.ResponseSize, Valid: true},
		Logs:              sql.NullString{String: invResult.Logs, Valid: invResult.Logs != ""},
		Error:             sql.NullString{String: invResult.Error, Valid: invResult.Error != ""},
	})

	// Return the function's response
	for k, v := range invResult.Headers {
		c.Header(k, v)
	}
	c.Data(invResult.StatusCode, "application/json", invResult.Body)
	return nil
}

func (e *v1Invocations) listInvocations(c *gin.Context) error {
	limit := GetQueryInt(c, "limit", 50)
	offset := GetQueryInt(c, "offset", 0)

	invocations, err := e.Querier.ListInvocations(c.Request.Context(), database.ListInvocationsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return fmt.Errorf("failed to list invocations: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"invocations": invocations})
	return nil
}

func (e *v1Invocations) getInvocation(c *gin.Context) error {
	id := c.Param("id")
	if id == "" {
		return fmt.Errorf("%s: invocation id is required", apitools.MsgInvalidParameter)
	}

	invocation, err := e.Querier.GetInvocation(c.Request.Context(), id)
	if err != nil {
		return fmt.Errorf("invocation not found: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"invocation": invocation})
	return nil
}

func (e *v1Invocations) listFunctionInvocations(c *gin.Context) error {
	functionID := c.Param("id")
	if functionID == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	limit := GetQueryInt(c, "limit", 50)
	offset := GetQueryInt(c, "offset", 0)

	invocations, err := e.Querier.ListInvocationsByFunction(c.Request.Context(), database.ListInvocationsByFunctionParams{
		FunctionID: functionID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return fmt.Errorf("failed to list function invocations: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"invocations": invocations})
	return nil
}

func (e *v1Invocations) getFunctionStats(c *gin.Context) error {
	functionID := c.Param("id")
	if functionID == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	// TODO: Parse time window from query params
	// For now, use last 24 hours
	// since := time.Now().Add(-24 * time.Hour)

	// stats, err := e.Querier.GetInvocationStats(c.Request.Context(), functionID, since)
	// if err != nil {
	// 	return fmt.Errorf("failed to get function stats: %w", err)
	// }

	// apitools.Ok(c, &apitools.Body{"stats": stats})
	// return nil

	return fmt.Errorf(apitools.MsgNotImplemented)
}