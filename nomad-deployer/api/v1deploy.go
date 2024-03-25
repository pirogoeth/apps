package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/nomad-deployer/codeutil"
	"github.com/pirogoeth/apps/nomad-deployer/types"
)

type v1DeployEndpoints struct {
	*types.ApiContext
}

func (v1d *v1DeployEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.POST("/renderManifest", v1d.postV1RenderManifest)
	router.POST("/deploy", v1d.postV1Deploy)
}

type V1DeployRequest struct {
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

type V1DeployResponse struct {
	// UpstreamStatus is the HTTP status code of the upstream request, if applicable.
	UpstreamStatus int `json:"upstreamStatus,omitempty"`
	// Error is the error message, if applicable.
	Err string `json:"error,omitempty"`
	// Message is a human-readable message, if applicable.
	Message string `json:"message,omitempty"`
	// Status is a concise description of the response status.
	Status string `json:"status,omitempty"`
	// DeploymentResponse is Nomad's response to registering the job
	DeploymentResponse *nomadApi.JobRegisterResponse `json:"deploymentResponse,omitempty"`
	// Warnings is a list of deployment warnings
	Warnings []string `json:"warnings,omitempty"`
	// CreatedVolumes is a mapping of volumes that were created during the deployment
	CreatedVolumes map[string]*nomadApi.CSIVolume `json:"createdVolumes,omitempty"`
}

func (r *V1DeployResponse) Error() string {
	return r.Err
}

func (r *V1DeployResponse) WithWarnings(warnings ...string) *V1DeployResponse {
	r.Warnings = append(r.Warnings, warnings...)
	return r
}

var _ error = (*V1DeployResponse)(nil)

func (e *v1DeployEndpoints) fetchDeploymentManifest(c *gin.Context, req *V1DeployRequest) (*codeutil.DeploymentManifest, *V1DeployResponse) {
	repo, resp, err := e.Github.Repositories.Get(c, e.Config.Github.Namespace, req.Repository)
	if err != nil {
		return nil, &V1DeployResponse{
			UpstreamStatus: resp.StatusCode,
			Err:            err.Error(),
			Message:        "Couldn't fetch repository information from GitHub",
		}
	}

	ctx := context.WithoutCancel(c)

	deployRef, err := codeutil.GetCommitByRef(ctx, e.Github, repo, req.Ref)
	if err != nil {
		return nil, &V1DeployResponse{
			Err:     err.Error(),
			Message: fmt.Sprintf("Couldn't fetch commit by ref '%s' for repository '%s'", req.Ref, repo.GetFullName()),
		}
	}

	deploymentManifest, err := codeutil.GetDeploymentManifest(ctx, e.Github, repo, deployRef, req.Path, req.Variables)
	if err != nil {
		return nil, &V1DeployResponse{
			Err: err.Error(),
			Message: fmt.Sprintf(
				"Couldn't fetch deployment manifest from repository '%s' at path '%s' for commit '%s'",
				repo.GetFullName(),
				req.Path,
				deployRef.Commit.GetSHA(),
			),
		}
	}

	return deploymentManifest, nil
}

func (e *v1DeployEndpoints) postV1Deploy(c *gin.Context) {
	warnings := make([]string, 0)
	req := &V1DeployRequest{}
	if err := c.Bind(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &V1DeployResponse{
			Err:      ErrInvalidRequest,
			Warnings: warnings,
		})
		return
	}

	deploymentManifest, resp := e.fetchDeploymentManifest(c, req)
	if resp != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, resp.WithWarnings(warnings...))
		return
	}

	// match up volume manifests with the volumes that are defined in the deployment manifest
	// note that we should never delete a volume that is alongside a job but not mounted to any task groups
	mountedVolumes := make([]string, 0)
	for _, volume := range deploymentManifest.Volumes {
		definedButNotMounted := true
		for _, taskGroup := range deploymentManifest.Job.TaskGroups {
			for _, mountedVolume := range taskGroup.Volumes {
				if mountedVolume.Source == volume.Name {
					definedButNotMounted = false
					mountedVolumes = append(mountedVolumes, volume.Name)
				}
			}
		}

		if definedButNotMounted {
			warnings = append(warnings, fmt.Sprintf("Volume '%s' is defined but not mounted to any task group", volume.Name))
		}
	}

	createdVolumes := make(map[string]*nomadApi.CSIVolume)
	for _, volumeName := range mountedVolumes {
		volume, ok := deploymentManifest.Volumes[volumeName]
		if !ok {
			// Should this be a critical failure? That would exclude externally-defined volumes..
			warnings = append(warnings, fmt.Sprintf("Volume '%s' is mounted but not defined", volumeName))
		}

		// check if volume already exists
		_, _, err := e.Nomad.CSIVolumes().Info(volumeName, &nomadApi.QueryOptions{
			Namespace: *deploymentManifest.Job.Namespace,
		})
		if err == nil {
			logrus.Infof("Volume '%s' already exists for deployment %s", volumeName, *deploymentManifest.Job.Name)
			continue
		}

		volumeResp, _, err := e.Nomad.CSIVolumes().Create(volume, &nomadApi.WriteOptions{
			Namespace: *deploymentManifest.Job.Namespace,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, &V1DeployResponse{
				Err:      err.Error(),
				Message:  "Could not create volume with Nomad",
				Warnings: warnings,
			})
			return
		}
		createdVolumes[volume.Name] = volumeResp[0]

		logrus.Infof("Created volume '%s' in %s for deployment %s", volumeName, volumeResp[0].Namespace, *deploymentManifest.Job.Name)
	}

	registerResp, _, err := e.Nomad.Jobs().Register(deploymentManifest.Job, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, &V1DeployResponse{
			Err:      err.Error(),
			Message:  "Could not register deployment job with Nomad",
			Warnings: warnings,
		})
		return
	}

	c.JSON(200, &V1DeployResponse{
		Status:             "ok",
		Message:            "Deployment started",
		Warnings:           warnings,
		DeploymentResponse: registerResp,
		CreatedVolumes:     createdVolumes,
	})
}

func (e *v1DeployEndpoints) postV1RenderManifest(c *gin.Context) {
	req := &V1DeployRequest{}
	if err := c.Bind(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &V1DeployResponse{
			Err: ErrInvalidRequest,
		})
		return
	}

	deploymentManifest, resp := e.fetchDeploymentManifest(c, req)
	if resp != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, resp)
		return
	}

	c.JSON(200, gin.H{
		"manifest": deploymentManifest,
	})
}
