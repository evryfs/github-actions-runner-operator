package githubapi

import (
	"context"
	"github.com/google/go-github/v40/github"
	"github.com/gregjones/httpcache"
	"github.com/palantir/go-githubapp/githubapp"
)

//IRunnerAPI is a service towards GitHubs runners
type IRunnerAPI interface {
	GetRunners(ctx context.Context, organization string, repository string, token string) ([]*github.Runner, error)
	UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerID int64) error
	CreateRegistrationToken(ctx context.Context, organization string, repository string, token string) (*github.RegistrationToken, error)
}

type runnerAPI struct {
	clientCreator githubapp.ClientCreator
}

//NewRunnerAPI gets a new instance of the API.
func NewRunnerAPI() (runnerAPI, error) {
	config := githubapp.Config{
		V3APIURL: "https://api.github.com",
		V4APIURL: "https://api.github.com",
	}
	config.SetValuesFromEnv("")

	clientCreator, err := githubapp.NewDefaultCachingClientCreator(config,
		githubapp.WithClientUserAgent("evryfs/garo"),
		githubapp.WithClientCaching(true, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
	)

	return runnerAPI{
		clientCreator: clientCreator,
	}, err
}

func (r runnerAPI) getClient(ctx context.Context, organization string, token string) (*github.Client, error) {
	if token != "" {
		return r.clientCreator.NewTokenClient(token)
	}

	client, err := r.clientCreator.NewAppClient()
	if err != nil {
		return nil, err
	}

	installationsService := githubapp.NewInstallationsService(client)
	installation, err := installationsService.GetByOwner(ctx, organization)
	if err != nil {
		return nil, err
	}

	return r.clientCreator.NewInstallationClient(installation.ID)
}

// Return all runners for the org
func (r runnerAPI) GetRunners(ctx context.Context, organization string, repository string, token string) ([]*github.Runner, error) {
	client, err := r.getClient(ctx, organization, token)
	if err != nil {
		return nil, err
	}

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
	client, err := r.getClient(ctx, organization, token)
	if err != nil {
		return err
	}

	if repository != "" {
		_, err := client.Actions.RemoveRunner(ctx, organization, repository, runnerID)
		return err
	}
	_, err = client.Actions.RemoveOrganizationRunner(ctx, organization, runnerID)

	return err
}

func (r runnerAPI) CreateRegistrationToken(ctx context.Context, organization string, repository string, token string) (*github.RegistrationToken, error) {
	client, err := r.getClient(ctx, organization, token)
	if err != nil {
		return nil, err
	}

	if repository != "" {
		regToken, _, err := client.Actions.CreateRegistrationToken(ctx, organization, repository)
		return regToken, err
	}

	regToken, _, err := client.Actions.CreateOrganizationRegistrationToken(ctx, organization)
	return regToken, err
}
