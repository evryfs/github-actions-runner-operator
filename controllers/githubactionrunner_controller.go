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
	"errors"
	"fmt"
	garov1alpha1 "github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
	"github.com/evryfs/github-actions-runner-operator/controllers/githubapi"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v32/github"
	"github.com/imdario/mergo"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/thoas/go-funk"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

const poolLabel = "garo.tietoevry.com/pool"

// GithubActionRunnerReconciler reconciles a GithubActionRunner object
type GithubActionRunnerReconciler struct {
	util.ReconcilerBase
	Log       logr.Logger
	GithubAPI githubapi.IRunnerAPI
}

// IsValid validates the CR and returns false if it is not valid.
func (r *GithubActionRunnerReconciler) IsValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*garov1alpha1.GithubActionRunner)
	if !ok {
		return false, errors.New("not a GithubActionRunner object")
	}
	if instance.Spec.MaxRunners < instance.Spec.MinRunners {
		return false, errors.New("MaxRunners must be greater or equal to minRunners")
	}

	return true, nil
}

// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners/*,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// Reconcile is the main loop implementing the controller action
func (r *GithubActionRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("githubactionrunner", req.NamespacedName)
	reqLogger.Info("Reconciling GithubActionRunner")

	// Fetch the GithubActionRunner instance
	instance := &garov1alpha1.GithubActionRunner{}
	if err := r.GetClient().Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return r.manageOutcome(ctx, instance, err)
	}

	if ok, err := r.IsValid(instance); !ok {
		return r.manageOutcome(ctx, instance, err)
	}

	token, err := r.tokenForRef(ctx, instance)
	if err != nil {
		reqLogger.Error(err, "Error reading secret")
		return r.manageOutcome(ctx, instance, err)
	}

	allRunners, err := r.GithubAPI.GetRunners(instance.Spec.Organization, instance.Spec.Repository, token)
	if err != nil {
		reqLogger.Error(err, "error from github api")
		return r.manageOutcome(ctx, instance, err)
	}
	runners := funk.Filter(allRunners, func(r *github.Runner) bool {
		return strings.HasPrefix(r.GetName(), instance.Name)
	}).([]*github.Runner)
	busyRunners := funk.Filter(runners, func(r *github.Runner) bool {
		return r.GetBusy()
	}).([]*github.Runner)

	podList, err := r.listRelatedPods(ctx, instance, "")
	if err != nil {
		return r.manageOutcome(ctx, instance, err)
	}

	if len(podList.Items) != len(runners) {
		reqLogger.Info("Pods and runner API not in sync, returning early")
		return r.manageOutcome(ctx, instance, nil)
	}

	// if under desired minimum instances or pool is saturated, scale up
	if len(runners) < instance.Spec.MinRunners || (len(runners) == len(busyRunners) && len(runners) < instance.Spec.MaxRunners) {
		instance.Status.CurrentSize = len(podList.Items)
		scale := funk.MaxInt([]int{instance.Spec.MinRunners - len(runners), 1}).(int)
		reqLogger.Info("Scaling up", "numInstances", scale)
		if err := r.scaleUp(ctx, scale, instance, reqLogger); err != nil {
			return r.manageOutcome(ctx, instance, err)
		}
		instance.Status.CurrentSize += scale
		err = r.GetClient().Status().Update(ctx, instance)

		return r.manageOutcome(ctx, instance, err)
	} else if len(runners) > instance.Spec.MaxRunners || (len(runners)-len(busyRunners) > 1 && len(runners) > instance.Spec.MinRunners) {
		reqLogger.Info("Scaling down", "totalrunners at github", len(runners), "maxrunners in CR", instance.Spec.MaxRunners)
		busyRunnerNames := funk.Map(busyRunners, func(runner *github.Runner) string {
			return runner.GetName()
		}).([]string)

		podList, err := r.listRelatedPods(ctx, instance, corev1.PodRunning)
		if err != nil {
			return r.manageOutcome(ctx, instance, err)
		}

		for _, pod := range podList.Items {
			if !funk.Contains(busyRunnerNames, pod.GetName()) {
				err := r.DeleteResourceIfExists(ctx, &pod)
				if err == nil {
					r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Deleted pod %s/%s", pod.Namespace, pod.Name))
					instance.Status.CurrentSize--
					err := r.GetClient().Status().Update(ctx, instance)
					if err != nil {
						return r.manageOutcome(ctx, instance, err)
					}
				}

				//awful hack - else we get reconciled before pod status has been updated/removed and will delete too many
				time.Sleep(3 * time.Second)
				return r.manageOutcome(ctx, instance, err)
			}
		}
	}

	return r.manageOutcome(ctx, instance, nil)
}

func (r *GithubActionRunnerReconciler) manageOutcome(ctx context.Context, instance *garov1alpha1.GithubActionRunner, issue error) (reconcile.Result, error) {
	return r.ManageOutcomeWithRequeue(ctx, instance, issue, instance.Spec.GetReconciliationPeriod())
}

// SetupWithManager configures the controller by using the passed mgr
func (r *GithubActionRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create an index for pod status since we filter on it
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&garov1alpha1.GithubActionRunner{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *GithubActionRunnerReconciler) scaleUp(ctx context.Context, amount int, instance *garov1alpha1.GithubActionRunner, reqLogger logr.Logger) error {
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
		result, err := controllerutil.CreateOrUpdate(ctx, r.GetClient(), pod, func() error {
			pod.Spec = *instance.Spec.PodTemplateSpec.Spec.DeepCopy()
			pod.Annotations = instance.Spec.PodTemplateSpec.Annotations
			if err := mergo.Merge(&pod.Labels, instance.Spec.PodTemplateSpec.ObjectMeta.Labels, mergo.WithAppendSlice); err != nil {
				return err
			}

			return controllerutil.SetControllerReference(instance, pod, r.GetScheme())
		})
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name, "result", result)
		if err != nil {
			return err
		}
		r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Created pod %s/%s", pod.Namespace, pod.Name))
	}

	return nil
}

func (r *GithubActionRunnerReconciler) listRelatedPods(ctx context.Context, cr *garov1alpha1.GithubActionRunner, phase corev1.PodPhase) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		//would be safer with ownerref too, but whatever
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{poolLabel: cr.Name},
	}
	if phase != "" {
		opts = append(opts, client.MatchingFields{"status.phase": string(phase)})
	}
	err := r.GetClient().List(ctx, podList, opts...)
	return podList, err
}

func (r *GithubActionRunnerReconciler) tokenForRef(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (string, error) {
	var secret corev1.Secret
	err := r.GetClient().Get(ctx, client.ObjectKey{Name: cr.Spec.TokenRef.Name, Namespace: cr.Namespace}, &secret)

	if err != nil {
		return "", err
	}
	return string(secret.Data[cr.Spec.TokenRef.Key]), nil
}
