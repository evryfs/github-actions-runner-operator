package githubapi

import (
	"context"
	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

//IRunnerAPI is a service towards GitHubs runners
type IRunnerAPI interface {
	GetRunners(ctx context.Context, organization string, repository string, token string) ([]*github.Runner, error)
	UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerID int64) error
}

type runnerAPI struct {
}

//NewRunnerAPI gets a new instance of the API.
func NewRunnerAPI() runnerAPI {
	return runnerAPI{}
}

func getClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&(oauth2.Token{
		AccessToken: token,
	}))
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return client
}

// Return all runners for the org
func (r runnerAPI) GetRunners(ctx context.Context, organization string, repository string, token string) ([]*github.Runner, error) {
	client := getClient(ctx, token)
	var allRunners []*github.Runner
	opts := &github.ListOptions{PerPage: 30}

	for {
		var runners *github.Runners
		var response *github.Response
		var err error

		if repository != "" {
			runners, response, err = client.Actions.ListRunners(ctx, organization, repository, opts)
		} else {
			runners, response, err = client.Actions.ListOrganizationRunners(ctx, organization, opts)
		}
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

func (r runnerAPI) UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerID int64) error {
	client := getClient(ctx, token)
	if repository != "" {
		//removeToken, _, err := client.Actions.CreateRemoveToken(ctx, organization, repository)
		_, err := client.Actions.RemoveRunner(ctx, organization, repository, runnerID)
		return err
	} else {
		//removeToken, _, err := client.Actions.CreateOrganizationRemoveToken(ctx, organization)
		_, err := client.Actions.RemoveOrganizationRunner(ctx, organization, runnerID)
		return err
	}
}
