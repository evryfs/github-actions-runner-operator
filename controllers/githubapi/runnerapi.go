package githubapi

import (
	"context"
	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

//IRunnerAPI is a service towards GitHubs runners
type IRunnerAPI interface {
	GetRunners(organization string, repository string, token string) ([]*github.Runner, error)
	UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerId int64) error
}

type runnerAPI struct {
}

//NewRunnerAPI gets a new instance of the API.
func NewRunnerAPI() runnerAPI {
	return runnerAPI{}
}

func getClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&(oauth2.Token{
		AccessToken: token,
	}))
	tc := oauth2.NewClient(context.TODO(), ts)
	client := github.NewClient(tc)

	return client
}

// Return all runners for the org
func (r runnerAPI) GetRunners(organization string, repository string, token string) ([]*github.Runner, error) {
	client := getClient(token)
	var allRunners []*github.Runner
	opts := &github.ListOptions{PerPage: 30}

	for {
		var runners *github.Runners
		var response *github.Response
		var err error

		if repository != "" {
			runners, response, err = client.Actions.ListRunners(context.TODO(), organization, repository, opts)
		} else {
			runners, response, err = client.Actions.ListOrganizationRunners(context.TODO(), organization, opts)
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

func (r runnerAPI) UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerId int64) error {
	client := getClient(token)
	if repository != "" {
		//removeToken, _, err := client.Actions.CreateRemoveToken(ctx, organization, repository)
		_, err := client.Actions.RemoveRunner(ctx, organization, repository, runnerId)
		return err
	} else {
		//removeToken, _, err := client.Actions.CreateOrganizationRemoveToken(ctx, organization)
		_, err := client.Actions.RemoveOrganizationRunner(ctx, organization, runnerId)
		return err
	}
}
