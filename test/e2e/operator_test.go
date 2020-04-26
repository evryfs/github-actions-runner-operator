package e2e

import (
	"github.com/evryfs/github-actions-runner-operator/pkg/apis/garo/v1alpha1"
	"github.com/gophercloud/gophercloud/testhelper"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"

	goctx "context"

	"github.com/evryfs/github-actions-runner-operator/pkg/apis"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestCr(t *testing.T) {
	githubActionRunnerList := &v1alpha1.GithubActionRunnerList{}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, githubActionRunnerList)
	testhelper.AssertNoErr(t, err)

	ctx := framework.NewContext(t)
	defer ctx.Cleanup()

	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: time.Minute, RetryInterval: retryInterval})
	testhelper.AssertNoErr(t, err)

	// get namespace
	namespace, err := ctx.GetOperatorNamespace()
	testhelper.AssertNoErr(t, err)

	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	testhelper.AssertNoErr(t, err)

	githubActionRunner := &v1alpha1.GithubActionRunner{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      "test",
		},
		Spec: v1alpha1.GithubActionRunnerSpec{
			MinRunners:   1,
			MaxRunners:   1,
			Organization: "test",
			PodSpec: v1core.PodSpec{
				Containers: []v1core.Container{{
					Name:  "test",
					Image: "hello-world:latest",
				}},
			},
		},
	}

	err = framework.Global.Client.Create(goctx.TODO(), githubActionRunner, &framework.CleanupOptions{TestContext: ctx, RetryInterval: retryInterval, Timeout: timeout})
	testhelper.AssertNoErr(t, err)

}
