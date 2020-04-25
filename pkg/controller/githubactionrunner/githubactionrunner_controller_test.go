package githubactionrunner

import (
	"context"
	"github.com/evryfs/github-actions-runner-operator/pkg/apis/garo/v1alpha1"
	"github.com/evryfs/github-actions-runner-operator/pkg/githubapi"
	"github.com/gophercloud/gophercloud/testhelper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func (r mockRunner) GetOrgRunners(_ string, _ string) (githubapi.Runners, error) {
	return githubapi.Runners{
		TotalCount: 0,
		Runners:    nil,
	}, nil
}

type mockRunner struct{}

func TestGithubactionRunnerController(t *testing.T) {
	const namespace = "someNamespace"
	const name = "somerunner"
	const secretName = "someSecretName"

	runner := &v1alpha1.GithubActionRunner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"label-key": "label-value",
			},
		},
		Spec: v1alpha1.GithubActionRunnerSpec{
			Organization: "someOrg",
			MinRunners:   1,
			MaxRunners:   1,
			PodSpec:      v1.PodSpec{},
			TokenRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: "someKey",
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
			"GH_TOKEN": []byte("someToken"),
		},
		StringData: nil,
		Type:       "Opaque",
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{runner, secret}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, runner)

	cl := fake.NewFakeClientWithScheme(s, objs...)

	r := &ReconcileGithubActionRunner{client: cl, scheme: s, githubApi: &mockRunner{}}

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
	testhelper.AssertEquals(t, runner.Spec.MinRunners, uint(len(podList.Items)))
}
