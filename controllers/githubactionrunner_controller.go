/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/evryfs/github-actions-runner-operator/controllers/githubapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	garov1alpha1 "github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
)

// GithubActionRunnerReconciler reconciles a GithubActionRunner object
type GithubActionRunnerReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	GithubApi githubapi.IRunnerApi
}

// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
func (r *GithubActionRunnerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("githubactionrunner", req.NamespacedName)

	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling GithubActionRunner")

	// Fetch the GithubActionRunner instance
	instance := &garov1alpha1.GithubActionRunner{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
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

	runners, err := r.GithubApi.GetRunners(instance.Spec.Organization, instance.Spec.Repository, token)

	if err != nil {
		reqLogger.Error(err, "error from github api")
		return reconcile.Result{}, err
	}

	if len(runners) < instance.Spec.MinRunners {
		podList, err := r.listRelatedPods(instance)
		if err == nil && len(podList.Items) == len(runners) { // all have settled/registered
			return r.scaleUp(instance.Spec.MinRunners-len(runners), instance, reqLogger)
		}
	} else if len(runners) > instance.Spec.MaxRunners {
		reqLogger.Info("Total runners over max, scaling down...", "totalrunners at github", len(runners), "maxrunners in CR", instance.Spec.MaxRunners)
		pods, err := r.listRelatedPods(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		for _, pod := range pods.Items[0 : len(runners)-instance.Spec.MaxRunners] {
			err = r.Client.Delete(context.TODO(), &pod, &client.DeleteOptions{})
			if err != nil {
				r.Log.Error(err, "Error deleting pod")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *GithubActionRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garov1alpha1.GithubActionRunner{}).
		Complete(r)
}

func (r *GithubActionRunnerReconciler) scaleUp(amount int, instance *garov1alpha1.GithubActionRunner, reqLogger logr.Logger) (reconcile.Result, error) {
	for i := 0; i < amount; i++ {
		pod := newPodForCR(instance)
		result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, pod, func() error {
			pod.Spec = *instance.Spec.PodSpec.DeepCopy()
			return controllerutil.SetControllerReference(instance, pod, r.Scheme)
		})
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name, "result", result)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Pod created successfully - don't requeue
	return reconcile.Result{}, nil
}

func (r *GithubActionRunnerReconciler) listRelatedPods(cr *garov1alpha1.GithubActionRunner) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		//would be safer with ownerref too, but whatever
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{"app": cr.Name},
		//client.MatchingFields{"status.phase": "Running"},
	}
	err := r.Client.List(context.TODO(), podList, opts...)
	return podList, err
}

func (r *GithubActionRunnerReconciler) tokenForRef(cr *garov1alpha1.GithubActionRunner) (string, error) {
	var token string
	var secret corev1.Secret
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: cr.Spec.TokenRef.Name, Namespace: cr.Namespace}, &secret)

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
	}
}
