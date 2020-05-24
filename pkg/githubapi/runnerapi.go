package githubapi

import (
	"context"
	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
)

type IRunnerApi interface {
	GetOrgRunners(organization string, token string) ([]*github.Runner, error)
}

type RunnerApi struct {
}

func NewRunnerApi() RunnerApi {
	return RunnerApi{}
}

func (r RunnerApi) GetOrgRunners(organization string, token string) ([]*github.Runner, error) {
	ts := oauth2.StaticTokenSource(&(oauth2.Token{
		AccessToken: token,
	}))
	tc := oauth2.NewClient(context.TODO(), ts)
	client := github.NewClient(tc)

	var allRunners []*github.Runner
	opts := &github.ListOptions{PerPage: 30}

	for {
		runners, response, err := client.Actions.ListOrganizationRunners(context.TODO(), organization, opts)
		if err != nil {
			return allRunners, err
		}
		allRunners = append(allRunners, runners.Runners...)
		if response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	return allRunners, nil
}
