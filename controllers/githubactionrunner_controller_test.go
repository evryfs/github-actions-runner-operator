package controllers

import (
	"context"
	"github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
	"github.com/google/go-github/v40/github"
	"github.com/gophercloud/gophercloud/testhelper"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func (r *mockAPI) GetRunners(ctx context.Context, organization string, repository string, token string) ([]*github.Runner, error) {
	args := r.Called(organization, repository, token)
	return args.Get(0).([]*github.Runner), args.Error(1)
}

func (r *mockAPI) UnregisterRunner(ctx context.Context, organization string, repository string, token string, runnerID int64) error {
	return nil
}

func (r *mockAPI) CreateRegistrationToken(ctx context.Context, organization string, repository string, token string) (*github.RegistrationToken, error) {
	return &github.RegistrationToken{
		Token:     github.String("sometoken"),
		ExpiresAt: &github.Timestamp{},
	}, nil
}

type mockAPI struct {
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

	const someLabel = "someLabel"
	const someLabelValue = "someLabelValue"

	var mockResult []*github.Runner

	mockAPI := new(mockAPI)
	mockAPI.On("GetRunners", org, repo, token).Return(mockResult, nil).Once()

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
			Repository:   repo,
			MinRunners:   2,
			MaxRunners:   2,
			PodTemplateSpec: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						someLabel: someLabelValue,
					},
					Annotations: map[string]string{
						"someAnnotationKey": "someAnnotationValue",
					},
				},
			},
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
	ctx := context.TODO()

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeBuilder.GroupVersion, runner)

	cl := fake.NewFakeClientWithScheme(s, objs...)

	fakeRecorder := record.NewFakeRecorder(10)
	r := &GithubActionRunnerReconciler{ReconcilerBase: util.NewReconcilerBase(cl, s, nil, fakeRecorder, nil), Log: zap.New(), GithubAPI: mockAPI}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}

	res, err := r.Reconcile(ctx, req)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, false, res.Requeue)

	podList := &v1.PodList{}
	err = r.GetClient().List(ctx, podList)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, runner.Spec.MinRunners, len(podList.Items))
	numEvents := len(fakeRecorder.Events)
	testhelper.AssertEquals(t, runner.Spec.MinRunners, numEvents)

	expectedLabels := map[string]string{
		someLabel: someLabelValue,
		poolLabel: name,
	}

	podObjectMeta := podList.Items[0].GetObjectMeta()
	testhelper.AssertDeepEquals(t, expectedLabels, podObjectMeta.GetLabels())
	testhelper.AssertDeepEquals(t, runner.Spec.PodTemplateSpec.GetObjectMeta().GetAnnotations(), podObjectMeta.GetAnnotations())

	// then scale down
	mockResult = append(mockResult, &github.Runner{
		ID:     pointer.Int64Ptr(1),
		Name:   pointer.StringPtr(podList.Items[0].Name),
		OS:     pointer.StringPtr("Linux"),
		Status: pointer.StringPtr("online"),
		Busy:   pointer.BoolPtr(false),
	}, &github.Runner{
		ID:     pointer.Int64Ptr(2),
		Name:   pointer.StringPtr(podList.Items[1].Name),
		OS:     pointer.StringPtr("Linux"),
		Status: pointer.StringPtr("online"),
		Busy:   pointer.BoolPtr(false),
	})
	mockAPI.On("GetRunners", org, repo, token).Return(mockResult, nil).Once()

	err = r.GetClient().Get(ctx, req.NamespacedName, runner)
	testhelper.AssertNoErr(t, err)
	runner.Spec.MinRunners = 1
	err = r.GetClient().Update(ctx, runner)
	testhelper.AssertNoErr(t, err)

	res, err = r.Reconcile(ctx, req)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, false, res.Requeue)

	podList = &v1.PodList{}
	err = r.GetClient().List(ctx, podList)
	testhelper.AssertNoErr(t, err)
	testhelper.AssertEquals(t, runner.Spec.MinRunners, len(podList.Items))
	testhelper.AssertEquals(t, numEvents+1, len(fakeRecorder.Events))
	mockAPI.AssertExpectations(t)
}
