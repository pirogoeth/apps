package codeutil

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v58/github"
	"github.com/pirogoeth/apps/pkg/goro"
)

type DeployableRefType int

func (t DeployableRefType) String() string {
	switch t {
	case DeployableRefTypeBranch:
		return "branch"
	case DeployableRefTypeReleaseTag:
		return "tag"
	case DeployableRefTypeSHA:
		return "sha"
	case DeployableRefTypePR:
		return "pr"
	default:
		return "unknown"
	}
}

const (
	DeployableRefTypeBranch DeployableRefType = iota
	DeployableRefTypeReleaseTag
	DeployableRefTypeSHA
	DeployableRefTypePR
)

// DeployableRef describes an resolved reference to a snapshot of code and the source of the ref
type DeployableRef struct {
	// Ref is the source ref string
	Ref string
	// Commit is the fetched commit
	Commit *github.RepositoryCommit
	// Source describes the source type the ref was loaded from
	Source DeployableRefSource
}

// DeployableRefSource describes the source of a deployable ref
type DeployableRefSource struct {
	// Type is the given (or discovered!) ref type
	Type DeployableRefType
	// Branch is the branch object if the ref is a branch
	Branch *github.Branch
	// Commit is the commit object if the ref is a commit
	Commit *github.RepositoryCommit
	// Tag is the tag object if the ref is a tag
	ReleaseTag *github.RepositoryRelease
	// PR is the pull request object if the ref is a pull request
	PR *github.PullRequest
}

// GetCommitByRef fetches a commit by a ref string.
func GetCommitByRef(ctx context.Context, githubClient *github.Client, repo *github.Repository, ref string) (*DeployableRef, error) {
	if strings.Contains(ref, ":") {
		parts := strings.SplitN(ref, ":", 2)
		refType, refName := strings.ToLower(parts[0]), parts[1]

		var deployRef *DeployableRef
		var err error

		switch refType {
		case "branch":
			deployRef, err = fetchBranch(ctx, githubClient, repo, refName)
		case "tag":
			deployRef, err = fetchTag(ctx, githubClient, repo, refName)
		case "sha":
			deployRef, err = fetchCommit(ctx, githubClient, repo, refName)
		case "pr":
			deployRef, err = fetchPullRequest(ctx, githubClient, repo, refName)
		default:
			return nil, fmt.Errorf("invalid ref type: %s", refType)
		}

		if err != nil {
			return nil, fmt.Errorf("while trying to fetch %s ref '%s': %w", refType, refName, err)
		}

		return deployRef, nil
	} else {
		rg := goro.NewRaceGroup[*DeployableRef](goro.RaceSuccess)
		rg.Add(func(ctx context.Context) (*DeployableRef, error) {
			return fetchBranch(ctx, githubClient, repo, ref)
		})
		rg.Add(func(ctx context.Context) (*DeployableRef, error) {
			return fetchTag(ctx, githubClient, repo, ref)
		})
		rg.Add(func(ctx context.Context) (*DeployableRef, error) {
			return fetchCommit(ctx, githubClient, repo, ref)
		})
		rg.Add(func(ctx context.Context) (*DeployableRef, error) {
			return fetchPullRequest(ctx, githubClient, repo, ref)
		})
		deployCommit, err := rg.Race(ctx)
		if deployCommit == nil {
			return nil, fmt.Errorf("while trying to fetch ref '%s': %w", ref, err)
		}

		return deployCommit, nil
	}
}

func fetchBranch(ctx context.Context, githubClient *github.Client, repo *github.Repository, ref string) (*DeployableRef, error) {
	branch, _, err := githubClient.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), ref, 3)
	if err != nil {
		return nil, fmt.Errorf("trying to fetch branch '%s': %w", ref, err)
	}

	commit := branch.GetCommit()

	return &DeployableRef{
		Ref:    ref,
		Commit: commit,
		Source: DeployableRefSource{
			Type:   DeployableRefTypeBranch,
			Branch: branch,
		},
	}, nil
}

func fetchTag(ctx context.Context, githubClient *github.Client, repo *github.Repository, ref string) (*DeployableRef, error) {
	release, _, err := githubClient.Repositories.GetReleaseByTag(ctx, repo.GetOwner().GetLogin(), repo.GetName(), ref)
	if err != nil {
		return nil, fmt.Errorf("trying to fetch tag '%s': %w", ref, err)
	}

	commit, err := fetchCommit(ctx, githubClient, repo, release.GetTargetCommitish())
	if err != nil {
		return nil, fmt.Errorf("trying to fetch commit for release tag '%s': %w", ref, err)
	}

	return &DeployableRef{
		Ref:    ref,
		Commit: commit.Commit,
		Source: DeployableRefSource{
			Type:       DeployableRefTypeReleaseTag,
			ReleaseTag: release,
		},
	}, nil
}

func fetchCommit(ctx context.Context, githubClient *github.Client, repo *github.Repository, ref string) (*DeployableRef, error) {
	commit, _, err := githubClient.Repositories.GetCommit(ctx, repo.GetOwner().GetLogin(), repo.GetName(), ref, nil)
	if err != nil {
		return nil, fmt.Errorf("trying to fetch commit for tag '%s': %w", ref, err)
	}

	return &DeployableRef{
		Ref:    ref,
		Commit: commit,
		Source: DeployableRefSource{
			Type:   DeployableRefTypeSHA,
			Commit: commit,
		},
	}, nil
}

func fetchPullRequest(ctx context.Context, githubClient *github.Client, repo *github.Repository, ref string) (*DeployableRef, error) {
	pullReqNum, err := strconv.Atoi(ref)
	if err != nil {
		return nil, fmt.Errorf("ref '%s' is not a valid numeric pull request ID: %w", ref, err)
	}

	pullReq, _, err := githubClient.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pullReqNum)
	if err != nil {
		return nil, fmt.Errorf("trying to fetch pull request '%s': %w", ref, err)
	}

	head := pullReq.GetHead()
	if head.GetRepo().GetID() != repo.GetID() {
		return nil, fmt.Errorf(
			"pull request '%s' is not from the same repository (remote head owned by %s)",
			ref,
			head.GetRepo().GetOwner().GetLogin(),
		)
	}

	commit, err := fetchCommit(ctx, githubClient, repo, head.GetSHA())
	if err != nil {
		return nil, fmt.Errorf("trying to fetch head commit for pull request '%s': %w", ref, err)
	}

	return &DeployableRef{
		Ref:    ref,
		Commit: commit.Commit,
		Source: DeployableRefSource{
			Type: DeployableRefTypePR,
			PR:   pullReq,
		},
	}, nil
}
