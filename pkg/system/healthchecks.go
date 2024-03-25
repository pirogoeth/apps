package system

import (
	"context"
	"fmt"

	"github.com/google/go-github/v58/github"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"
)

func NomadClientHealthCheck(_ context.Context, nomadClient *nomadApi.Client) error {
	leader, err := nomadClient.Status().Leader()
	if err != nil {
		return fmt.Errorf("could not check nomad leader: %w", err)
	}

	logrus.Infof("nomad-client.ready: leader=%s", leader)
	return nil
}

func RegisterNomadClientReadiness(nomadClient *nomadApi.Client) {
	check := func(ctx context.Context) (string, error) {
		return "ready, leader found!", NomadClientHealthCheck(ctx, nomadClient)
	}

	AddReadinessCheck("nomad-client", check)
}

func GithubClientHealthCheck(ctx context.Context, ghClient *github.Client) error {
	currentUser, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("could not check github client user: %w", err)
	}

	logrus.Infof("github-client.ready: currentUser='%s'", *currentUser.Login)
	return nil
}

func RegisterGithubClientReadiness(ghClient *github.Client) {
	check := func(ctx context.Context) (string, error) {
		return "ready, logged in!", GithubClientHealthCheck(ctx, ghClient)
	}

	AddReadinessCheck("github-client", check)
}
