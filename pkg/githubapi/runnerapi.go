package githubapi

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// types from api
type Runner struct {
	Id     uint   `json:"id"`
	Name   string `json:"name"`
	Os     string `json:"os"`
	Status string `json:"status"`
}

type Runners struct {
	TotalCount int      `json:"total_count"`
	Runners    []Runner `json:"runners"`
}

func (r Runners) ByStatus(status string) []Runner {
	filtered := make([]Runner, 0)
	for _, val := range r.Runners {
		if strings.EqualFold(val.Status, status) {
			filtered = append(filtered, val)
		}
	}

	return filtered
}

type IRunnerApi interface {
	GetOrgRunners(organization string, token string) (Runners, error)
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type RunnerApi struct {
	client HttpClient
	url    string
}

func NewRunnerApi(client HttpClient, baseUrl string) RunnerApi {
	return RunnerApi{
		client: client,
		url:    baseUrl,
	}
}

func DefaultRunnerAPI() RunnerApi {
	return NewRunnerApi(&http.Client{}, "https://api.github.com")
}

func (r RunnerApi) GetOrgRunners(organization string, token string) (Runners, error) {
	var runners Runners

	request, err := http.NewRequest(http.MethodGet,
		r.url+"/orgs/"+organization+"/actions/runners", nil)
	if err != nil {
		return runners, err
	}

	request.Header.Add("Authorization", "token "+token)
	response, err := r.client.Do(request)

	if err != nil {
		return runners, err
	} else if response.StatusCode != 200 {
		return runners, errors.New("bad response-code")
	} else {
		defer response.Body.Close()
		bytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return runners, err
		}
		err = json.Unmarshal(bytes, &runners)
		if err != nil {
			return runners, err
		}
	}

	return runners, err
}
