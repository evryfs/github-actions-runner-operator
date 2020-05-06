package githubactionrunner

import (
	"context"
	"github.com/evryfs/github-actions-runner-operator/pkg/githubapi"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	garov1alpha1 "github.com/evryfs/github-actions-runner-operator/pkg/apis/garo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_githubactionrunner")

// Add creates a new GithubActionRunner Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGithubActionRunner{client: mgr.GetClient(), scheme: mgr.GetScheme(), githubApi: githubapi.DefaultRunnerAPI()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("githubactionrunner-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GithubActionRunner
	err = c.Watch(&source.Kind{Type: &garov1alpha1.GithubActionRunner{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner GithubActionRunner
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &garov1alpha1.GithubActionRunner{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGithubActionRunner implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGithubActionRunner{}

// ReconcileGithubActionRunner reconciles a GithubActionRunner object
type ReconcileGithubActionRunner struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	githubApi githubapi.IRunnerApi
}

// Reconcile reads that state of the cluster for a GithubActionRunner object and makes changes based on the state read
// and what is in the GithubActionRunner.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGithubActionRunner) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GithubActionRunner")

	// Fetch the GithubActionRunner instance
	instance := &garov1alpha1.GithubActionRunner{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	token, err := r.tokenForRef(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	runners, err := r.githubApi.GetOrgRunners(instance.Spec.Organization, token)

	if err != nil {
		reqLogger.Error(err, "error from github api")
		return reconcile.Result{}, err
	}

	if runners.TotalCount < instance.Spec.MinRunners {
		podList, err := r.listRelatedPods(instance)
		if err == nil && len(podList.Items) == runners.TotalCount { // all have settled/registered
			return r.scaleUp(instance.Spec.MinRunners-runners.TotalCount, instance, reqLogger)
		}
	} else if runners.TotalCount > instance.Spec.MaxRunners {
		reqLogger.Info("Total runners over max, scaling down...", "totalrunners at github", runners.TotalCount, "maxrunners in CR", instance.Spec.MaxRunners)
		pods, err := r.listRelatedPods(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		for _, pod := range pods.Items[0 : runners.TotalCount-instance.Spec.MaxRunners] {
			err = r.client.Delete(context.TODO(), &pod, &client.DeleteOptions{})
			if err != nil {
				log.Error(err, "Error deleting pod")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGithubActionRunner) scaleUp(amount int, instance *garov1alpha1.GithubActionRunner, reqLogger logr.Logger) (reconcile.Result, error) {
	for i := 0; i < amount; i++ {
		pod := newPodForCR(instance)

		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err := r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Pod created successfully - don't requeue
	return reconcile.Result{}, nil
}

func (r *ReconcileGithubActionRunner) listRelatedPods(cr *garov1alpha1.GithubActionRunner) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		//would be safer with ownerref too, but whatever
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{"app": cr.Name},
		//client.MatchingFields{"status.phase": "Running"},
	}
	err := r.client.List(context.TODO(), podList, opts...)
	return podList, err
}

func (r *ReconcileGithubActionRunner) tokenForRef(cr *garov1alpha1.GithubActionRunner) (string, error) {
	var token string
	var secret corev1.Secret
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: cr.Spec.TokenRef.Name, Namespace: cr.Namespace}, &secret)

	if err != nil {
		return token, err
	}
	token = string(secret.Data[cr.Spec.TokenRef.Key])

	return token, err
}

// newPodForCR returns a pod with the same name pattern and namespace as the cr, based on the cr's podspec
func newPodForCR(cr *garov1alpha1.GithubActionRunner) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cr.Name + "-pod-",
			Namespace:    cr.Namespace,
			Labels:       labels,
		},
		Spec: *cr.Spec.PodSpec.DeepCopy(),
	}
}
