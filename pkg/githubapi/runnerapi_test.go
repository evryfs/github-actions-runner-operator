package githubapi

import (
	"encoding/json"
	"github.com/gophercloud/gophercloud/testhelper"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrgRunners(t *testing.T) {
	const orgName = "someOrg"
	const token = "someToken"
	expected := Runners{
		TotalCount: 1,
		Runners: []Runner{
			{
				Id:     0,
				Name:   "SomeRunnerName",
				Os:     "Linux",
				Status: "online",
			}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		testhelper.AssertEquals(t, req.URL.String(), "/orgs/"+orgName+"/actions/runners")
		testhelper.AssertEquals(t, req.Header.Get("Authorization"), "token "+token)

		bytes, err := json.Marshal(expected)
		testhelper.AssertNoErr(t, err)

		_, err = rw.Write(bytes)
		testhelper.AssertNoErr(t, err)
	}))
	defer server.Close()

	result, err := NewRunnerApi(server.Client(), server.URL).GetOrgRunners(orgName, token)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertDeepEquals(t, expected, result)
}
