package api

import (
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

// Adapters to convert between database types and compute types

func dbFunctionToComputeFunction(dbFunc database.Function) *compute.Function {
	return &compute.Function{
		ID:             dbFunc.ID,
		Name:           dbFunc.Name,
		Description:    dbFunc.Description.String,
		CodePath:       dbFunc.CodePath,
		Runtime:        dbFunc.Runtime,
		Handler:        dbFunc.Handler,
		TimeoutSeconds: int32(dbFunc.TimeoutSeconds),
		MemoryMB:       int32(dbFunc.MemoryMb),
		EnvVars:        dbFunc.EnvVars.String,
	}
}

func dbDeploymentToComputeDeployment(dbDep database.Deployment) *compute.Deployment {
	return &compute.Deployment{
		ID:         dbDep.ID,
		FunctionID: dbDep.FunctionID,
		Provider:   dbDep.Provider,
		ResourceID: dbDep.ResourceID,
		Status:     dbDep.Status,
		Replicas:   int32(dbDep.Replicas),
		ImageTag:   dbDep.ImageTag.String,
	}
}

func computeDeployResultToTypesDeployResult(compResult *compute.DeployResult) *types.DeployResult {
	return &types.DeployResult{
		DeploymentID: compResult.DeploymentID,
		ResourceID:   compResult.ResourceID,
		ImageTag:     compResult.ImageTag,
	}
}

func typesInvocationRequestToComputeInvocationRequest(typesReq *types.InvocationRequest) *compute.InvocationRequest {
	return &compute.InvocationRequest{
		FunctionID: typesReq.FunctionID,
		Body:       typesReq.Body,
		Headers:    typesReq.Headers,
		Method:     typesReq.Method,
		Path:       typesReq.Path,
		QueryArgs:  typesReq.QueryArgs,
	}
}

func computeInvocationResultToTypesInvocationResult(compResult *compute.InvocationResult) *types.InvocationResult {
	return &types.InvocationResult{
		StatusCode:   compResult.StatusCode,
		Body:         compResult.Body,
		Headers:      compResult.Headers,
		DurationMS:   compResult.DurationMS,
		MemoryUsedMB: compResult.MemoryUsedMB,
		ResponseSize: compResult.ResponseSize,
		Logs:         compResult.Logs,
		Error:        compResult.Error,
	}
}