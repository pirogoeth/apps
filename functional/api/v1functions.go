package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pirogoeth/apps/pkg/apitools"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

type v1Functions struct {
	*types.ApiContext
}

func (e *v1Functions) RegisterRoutesTo(router *gin.RouterGroup) {
	functions := router.Group("/functions")
	
	functions.POST("", apitools.ErrorWrapEndpoint(e.createFunction))
	functions.GET("", apitools.ErrorWrapEndpoint(e.listFunctions))
	functions.GET("/:id", apitools.ErrorWrapEndpoint(e.getFunction))
	functions.PUT("/:id", apitools.ErrorWrapEndpoint(e.updateFunction))
	functions.DELETE("/:id", apitools.ErrorWrapEndpoint(e.deleteFunction))
	functions.POST("/:id/deploy", apitools.ErrorWrapEndpoint(e.deployFunction))
}

func (e *v1Functions) createFunction(c *gin.Context) error {
	var req types.CreateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return fmt.Errorf("%s: %w", apitools.MsgFailedToBind, err)
	}

	// Validate request
	if req.Name == "" {
		return fmt.Errorf("%s: function name is required", apitools.MsgInvalidParameter)
	}
	if req.Runtime == "" {
		return fmt.Errorf("%s: runtime is required", apitools.MsgInvalidParameter)
	}
	if req.Handler == "" {
		return fmt.Errorf("%s: handler is required", apitools.MsgInvalidParameter)
	}

	// Generate function ID
	functionID := uuid.New().String()

	// Set defaults
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 30
	}
	memoryMB := req.MemoryMB
	if memoryMB == 0 {
		memoryMB = 128
	}

	// Store function code to filesystem
	codePath, err := e.storeFunctionCode(functionID, req.Code, req.Runtime)
	if err != nil {
		return fmt.Errorf("failed to store function code: %w", err)
	}

	// Serialize environment variables
	envVarsJSON := "{}"
	if len(req.EnvVars) > 0 {
		envBytes, err := json.Marshal(req.EnvVars)
		if err != nil {
			return fmt.Errorf("failed to serialize environment variables: %w", err)
		}
		envVarsJSON = string(envBytes)
	}

	// Create database record
	function, err := e.Querier.CreateFunction(c.Request.Context(), database.CreateFunctionParams{
		ID:             functionID,
		Name:           req.Name,
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		CodePath:       codePath,
		Runtime:        req.Runtime,
		Handler:        req.Handler,
		TimeoutSeconds: int64(timeoutSeconds),
		MemoryMb:       int64(memoryMB),
		EnvVars:        sql.NullString{String: envVarsJSON, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"function": function})
	return nil
}

func (e *v1Functions) listFunctions(c *gin.Context) error {
	functions, err := e.Querier.ListFunctions(c.Request.Context())
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"functions": functions})
	return nil
}

func (e *v1Functions) getFunction(c *gin.Context) error {
	id := c.Param("id")
	if id == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	function, err := e.Querier.GetFunction(c.Request.Context(), id)
	if err != nil {
		return fmt.Errorf("function not found: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"function": function})
	return nil
}

func (e *v1Functions) updateFunction(c *gin.Context) error {
	id := c.Param("id")
	if id == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	var req types.UpdateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return fmt.Errorf("%s: %w", apitools.MsgFailedToBind, err)
	}

	// TODO: Implement function update
	// 1. Validate function exists
	// 2. Update function code if provided
	// 3. Update database record
	// 4. Trigger redeployment if needed
	// 5. Return updated function details

	return fmt.Errorf(apitools.MsgNotImplemented)
}

func (e *v1Functions) deleteFunction(c *gin.Context) error {
	id := c.Param("id")
	if id == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	// TODO: Implement function deletion
	// 1. Validate function exists
	// 2. Stop and remove deployments
	// 3. Clean up function code
	// 4. Delete database record

	err := e.Querier.DeleteFunction(c.Request.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}

	c.JSON(http.StatusNoContent, nil)
	return nil
}

// Helper method to store function code
func (e *v1Functions) storeFunctionCode(functionID, codeBase64, runtime string) (string, error) {
	// Decode base64 code
	codeBytes, err := base64.StdEncoding.DecodeString(codeBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 code: %w", err)
	}

	// Create function directory
	functionsDir := e.Config.Storage.FunctionsPath
	functionDir := filepath.Join(functionsDir, functionID)
	
	if err := os.MkdirAll(functionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create function directory: %w", err)
	}

	// Store code (for now, just save as code.zip)
	codePath := filepath.Join(functionDir, "code.zip")
	if err := os.WriteFile(codePath, codeBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write function code: %w", err)
	}

	return codePath, nil
}

func (e *v1Functions) deployFunction(c *gin.Context) error {
	id := c.Param("id")
	if id == "" {
		return fmt.Errorf("%s: function id is required", apitools.MsgInvalidParameter)
	}

	// Validate function exists
	function, err := e.Querier.GetFunction(c.Request.Context(), id)
	if err != nil {
		return fmt.Errorf("function not found: %w", err)
	}

	// Get active compute provider
	provider, err := e.Compute.Get(e.Config.Compute.Provider)
	if err != nil {
		return fmt.Errorf("compute provider not available: %w", err)
	}

	// Convert function to compute type
	computeFunc := dbFunctionToComputeFunction(function)

	// Deploy to compute provider
	deployResult, err := provider.Deploy(c.Request.Context(), computeFunc, "")
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	compResult, ok := deployResult.(*compute.DeployResult)
	if !ok {
		return fmt.Errorf("invalid deploy result type")
	}

	// Convert back to types
	result := computeDeployResultToTypesDeployResult(compResult)

	// Create deployment record
	deployment, err := e.Querier.CreateDeployment(c.Request.Context(), database.CreateDeploymentParams{
		ID:         result.DeploymentID,
		FunctionID: id,
		Provider:   e.Config.Compute.Provider,
		ResourceID: result.ResourceID,
		Status:     string(types.DeploymentStatusActive),
		Replicas:   1,
		ImageTag:   sql.NullString{String: result.ImageTag, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment record: %w", err)
	}

	apitools.Ok(c, &apitools.Body{"deployment": deployment})
	return nil
}