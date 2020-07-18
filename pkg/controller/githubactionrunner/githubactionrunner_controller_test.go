package githubactionrunner

import (
	"context"
	"github.com/evryfs/github-actions-runner-operator/pkg/apis/garo/v1alpha1"
	"github.com/google/go-github/v31/github"
	"github.com/gophercloud/gophercloud/testhelper"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func (r *mockApi) GetRunners(organization string, repository string, token string) ([]*github.Runner, error) {
	args := r.Called(organization, repository, token)
	return args.Get(0).([]*github.Runner), nil
}

type mockApi struct {
	mock.Mock
}

func TestGithubactionRunnerController(t *testing.T) {
	const namespace = "someNamespace"
	const name = "somerunner"
	const secretName = "someSecretName"
	const org = "SomeOrg"
	const repo = ""
	const token = "someToken"
	const tokenKey = "GH_TOKEN"

	/*
		mockResult := []*github.Runner {{
			ID:     pointer.Int64Ptr(123),
			Name:   pointer.StringPtr("someName"),
			OS:     pointer.StringPtr("Linux"),
			Status: pointer.StringPtr("online"),
		},
		}
	*/

	var mockResult []*github.Runner
	mockApi := new(mockApi)
	mockApi.On("GetRunners", org, repo, token).Return(mockResult)

	runner := &v1alpha1.GithubActionRunner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"label-key": "label-value",
			},
		},
		Spec: v1alpha1.GithubActionRunnerSpec{
			Organization: org,
			Repository: repo,
			MinRunners:   2,
			MaxRunners:   2,
			PodSpec:      v1.PodSpec{},
			TokenRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: tokenKey,
			},
		},
	}

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Data: map[string][]byte{
			tokenKey: []byte(token),
		},
		StringData: nil,
		Type:       "Opaque",
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{runner, secret}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, runner)

	cl := fake.NewFakeClientWithScheme(s, objs...)

	r := &ReconcileGithubActionRunner{client: cl, scheme: s, githubApi: mockApi}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}

	res, err := r.Reconcile(req)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, false, res.Requeue)

	podList := &v1.PodList{}
	err = r.client.List(context.TODO(), podList)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, runner.Spec.MinRunners, len(podList.Items))
}
