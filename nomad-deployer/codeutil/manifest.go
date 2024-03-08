package codeutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/go-github/v58/github"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec2"
)

type DeploymentManifest struct {
	// Job is the parsed version of the Nomad job as loaded from the repo
	Job *api.Job
	// Ref is the source git ref of the deployment manifest
	Ref *DeployableRef
}

func GetDeploymentManifest(ctx context.Context, githubClient *github.Client, repo *github.Repository, deployRef *DeployableRef, path string, variables map[string]string) (*DeploymentManifest, error) {
	path = strings.TrimPrefix(path, "/")

	manifestPath := fmt.Sprintf("%s/.deployer/job.nomad.hcl", path)
	manifestFile, _, err := githubClient.Repositories.DownloadContents(
		ctx,
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		manifestPath,
		&github.RepositoryContentGetOptions{
			Ref: deployRef.Commit.GetSHA(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"could not fetch manifest content (at `%s`) for commit '%s' in repository '%s': %w",
			manifestPath,
			deployRef.Commit.GetSHA(),
			repo.GetFullName(),
			err,
		)
	}
	defer manifestFile.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, manifestFile)
	if err != nil {
		return nil, fmt.Errorf("could not copy manifest content to buffer: %w", err)
	}

	parseCfg := &jobspec2.ParseConfig{
		Path:       manifestPath,
		Body:       buf.Bytes(),
		AllowFS:    false,
		Strict:     true,
		Envs:       generateEnvVars(repo, deployRef, path),
		VarContent: generateVars(variables),
	}
	job, err := jobspec2.ParseWithConfig(parseCfg)
	if err != nil {
		return nil, fmt.Errorf(
			"could not parse manifest content (at `%s`) for commit '%s' in repository '%s': %w",
			manifestPath,
			deployRef.Commit.GetSHA(),
			repo.GetFullName(),
			err,
		)
	}

	return &DeploymentManifest{
		Job: job,
		Ref: deployRef,
	}, nil
}

func generateEnvVars(repo *github.Repository, deployRef *DeployableRef, path string) []string {
	envVars := map[string]string{
		"DEPLOY_MANIFEST_PATH":  path,
		"DEPLOY_REF":            deployRef.Ref,
		"DEPLOY_SHA":            deployRef.Commit.GetSHA(),
		"DEPLOY_TYPE":           deployRef.Source.Type.String(),
		"DEPLOY_REPO_NAME":      repo.GetName(),
		"DEPLOY_REPO_FULL_NAME": repo.GetFullName(),
		"DEPLOY_REPO_OWNER":     repo.GetOwner().GetLogin(),
		"DEPLOY_REPO_PRIVATE":   strconv.FormatBool(repo.GetPrivate()),
	}

	switch deployRef.Source.Type {
	case DeployableRefTypePR:
		key := "DEPLOY_PULL_REQUEST_"
		pr := deployRef.Source.PR

		envVars[key+"NUMBER"] = strconv.Itoa(pr.GetNumber())
		envVars[key+"STATE"] = pr.GetState()
		envVars[key+"DRAFT"] = strconv.FormatBool(pr.GetDraft())
		envVars[key+"CREATED_USER"] = pr.GetUser().GetLogin()
	case DeployableRefTypeBranch:
		key := "DEPLOY_BRANCH_"
		branch := deployRef.Source.Branch

		envVars[key+"NAME"] = branch.GetName()
		envVars[key+"HEAD_COMMIT_SHA"] = branch.GetCommit().GetSHA()
		envVars[key+"HEAD_COMMIT_AUTHOR"] = branch.GetCommit().GetAuthor().GetLogin()
		envVars[key+"HEAD_COMMIT_COMMITTER"] = branch.GetCommit().GetCommitter().GetLogin()
	case DeployableRefTypeReleaseTag:
		key := "DEPLOY_RELEASE_"
		tag := deployRef.Source.ReleaseTag

		envVars[key+"ID"] = strconv.FormatInt(tag.GetID(), 10)
		envVars[key+"NAME"] = tag.GetName()
		envVars[key+"TARGET_COMMITISH"] = tag.GetTargetCommitish()
		envVars[key+"AUTHOR"] = tag.GetAuthor().GetLogin()
		envVars[key+"DRAFT"] = strconv.FormatBool(tag.GetDraft())
		envVars[key+"PRERELEASE"] = strconv.FormatBool(tag.GetPrerelease())
	case DeployableRefTypeSHA:
		key := "DEPLOY_COMMIT_"
		sha := deployRef.Source.Commit

		envVars[key+"SHA"] = sha.GetSHA()
		envVars[key+"AUTHOR"] = sha.GetAuthor().GetLogin()
		envVars[key+"COMMITTER"] = sha.GetCommitter().GetLogin()
	}

	vars := make([]string, 0)
	for k, v := range envVars {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}

	return vars
}

func generateVars(variables map[string]string) (out string) {
	for k, v := range variables {
		out = out + fmt.Sprintf("%s = \"%s\"\n", k, v)
	}

	return
}
