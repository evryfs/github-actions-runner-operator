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
	"fmt"
	"github.com/evryfs/github-actions-runner-operator/controllers/githubapi"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v32/github"
	"github.com/imdario/mergo"
	"github.com/thoas/go-funk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	garov1alpha1 "github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
)

const poolLabel = "garo.tietoevry.com/pool"

// GithubActionRunnerReconciler reconciles a GithubActionRunner object
type GithubActionRunnerReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	GithubAPI githubapi.IRunnerAPI
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *GithubActionRunnerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("githubactionrunner", req.NamespacedName)
	reqLogger.Info("Reconciling GithubActionRunner")

	// Fetch the GithubActionRunner instance
	instance := &garov1alpha1.GithubActionRunner{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, instance); err != nil {
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
		reqLogger.Error(err, "Error reading secret")
		return reconcile.Result{}, err
	}

	allRunners, err := r.GithubAPI.GetRunners(instance.Spec.Organization, instance.Spec.Repository, token)
	if err != nil {
		reqLogger.Error(err, "error from github api")
		return reconcile.Result{}, err
	}
	runners := funk.Filter(allRunners, func(r *github.Runner) bool {
		return strings.HasPrefix(r.GetName(), instance.Name)
	}).([]*github.Runner)
	busyRunners := funk.Filter(runners, func(r *github.Runner) bool {
		return r.GetBusy()
	}).([]*github.Runner)

	result := reconcile.Result{RequeueAfter: instance.Spec.GetReconciliationPeriod()}

	// if under desired minimum instances or pool is saturated, scale up
	if len(runners) < instance.Spec.MinRunners || (len(runners) == len(busyRunners) && len(runners) < instance.Spec.MaxRunners) {
		podList, err := r.listRelatedPods(instance, "")
		instance.Status.CurrentSize = len(podList.Items)
		if err == nil && len(podList.Items) == len(runners) { // all have settled/registered
			scale := funk.MaxInt([]int{instance.Spec.MinRunners - len(runners), 1}).(int)
			reqLogger.Info("Scaling up", "numInstances", scale)
			if err := r.scaleUp(scale, instance, reqLogger); err != nil {
				return result, err
			}
			instance.Status.CurrentSize += scale
			err = r.Status().Update(context.Background(), instance)

			return result, err
		}
	} else if len(runners) > instance.Spec.MaxRunners || (len(runners)-len(busyRunners) > 1 && len(runners) > instance.Spec.MinRunners) {
		reqLogger.Info("Scaling down", "totalrunners at github", len(runners), "maxrunners in CR", instance.Spec.MaxRunners)
		pods, err := r.listRelatedPods(instance, corev1.PodRunning)
		if err != nil {
			return result, err
		}
		if len(pods.Items) != len(runners) {
			reqLogger.Info("Pods and runner API not in sync, returning early")
			return result, nil
		}

		busyRunnerNames := funk.Map(busyRunners, func(runner *github.Runner) string {
			return runner.GetName()
		}).([]string)

		for _, pod := range pods.Items {
			if !funk.Contains(busyRunnerNames, pod.GetName()) {
				err = r.Client.Delete(context.TODO(), &pod, &client.DeleteOptions{})
				if err == nil {
					instance.Status.CurrentSize--
					r.Recorder.Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Deleted pod %s/%s", pod.Namespace, pod.Name))
				}
				break
			}
		}

		defer r.Status().Update(context.TODO(), instance)
		return result, err
	}

	return result, err
}

func (r *GithubActionRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create an index for pod status since we filter on it
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj runtime.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&garov1alpha1.GithubActionRunner{}).
		Complete(r)
}

func (r *GithubActionRunnerReconciler) scaleUp(amount int, instance *garov1alpha1.GithubActionRunner, reqLogger logr.Logger) error {
	for i := 0; i < amount; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-pod-",
				Namespace:    instance.Namespace,
				Labels: map[string]string{
					poolLabel: instance.Name,
				},
			},
		}
		result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, pod, func() error {
			pod.Spec = *instance.Spec.PodTemplateSpec.Spec.DeepCopy()
			pod.Annotations = instance.Spec.PodTemplateSpec.Annotations
			if err := mergo.Merge(&pod.Labels, &instance.Spec.PodTemplateSpec.ObjectMeta.Labels); err != nil {
				return err
			}

			return controllerutil.SetControllerReference(instance, pod, r.Scheme)
		})
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name, "result", result)
		if err != nil {
			return err
		}
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Created pod %s/%s", pod.Namespace, pod.Name))
	}

	return nil
}

func (r *GithubActionRunnerReconciler) listRelatedPods(cr *garov1alpha1.GithubActionRunner, phase corev1.PodPhase) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		//would be safer with ownerref too, but whatever
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{poolLabel: cr.Name},
	}
	if phase != "" {
		opts = append(opts, client.MatchingFields{"status.phase": string(phase)})
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
