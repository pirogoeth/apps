package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/nomad-deployer/codeutil"
	"github.com/pirogoeth/apps/nomad-deployer/types"
)

type v1DeployEndpoints struct {
	*types.ApiContext
}

type PostV1DeploymentRequest struct {
	// Repository is the name of the repository to deploy, within the configured namespace
	Repository string `json:"repository"`
	// Ref is the version reference of the repository to deploy.
	// This can be a branch, tag, commit SHA, or pull request number.
	// If there are conflicting refs, use a prefix of the ref type to disambiguate:
	//   - `branch:` for branches
	//   - `tag:` for tags
	//   - `sha:` for commit SHAs
	//   - `pr:` for pull requests
	Ref string `json:"ref"`
	// Path is the path of the application to deploy within the repository.
	// If not set, defaults to `/`
	Path string `json:"path" default:"/"`
	// Canary is a map of app group names to instance counts.
	Canary map[string]int `json:"canary"`
	// Variables is a mapping of variables that should be injected into the deployment
	Variables map[string]string `json:"variables"`
}

func (e *v1DeployEndpoints) postV1Deploy(c *gin.Context) {
	req := &PostV1DeploymentRequest{}
	if err := c.Bind(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": ErrInvalidRequest,
		})
		return
	}

	repo, resp, err := e.Github.Repositories.Get(c, e.Config.Github.Namespace, req.Repository)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"upstream_status": resp.StatusCode,
			"error":           err.Error(),
		})
		return
	}

	ctx := context.WithoutCancel(c)

	deployRef, err := codeutil.GetCommitByRef(ctx, e.Github, repo, req.Ref)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("Couldn't fetch commit by ref '%s' for repository '%s'", req.Ref, repo.GetFullName()),
			"error":   err.Error(),
		})
		return
	}

	deploymentManifest, err := codeutil.GetDeploymentManifest(ctx, e.Github, repo, deployRef, req.Path, req.Variables)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf(
				"Couldn't fetch deployment manifest from repository '%s' at path '%s' for commit '%s'",
				repo.GetFullName(),
				req.Path,
				deployRef.Commit.GetSHA(),
			),
			"error": err.Error(),
		})
		return
	}

	logrus.Debugf("deployment manifest: %#v", deploymentManifest)

	c.JSON(200, gin.H{
		"status": "ok",
	})
}
