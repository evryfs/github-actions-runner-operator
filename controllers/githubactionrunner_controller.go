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
	"github.com/google/go-github/v33/github"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

const poolLabel = "garo.tietoevry.com/pool"
const finalizer = "garo.tietoevry.com/runner-registration"

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
	return instance.Spec.IsValid()
}

// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners/*,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// Reconcile is the main loop implementing the controller action
func (r *GithubActionRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("githubactionrunner", req.NamespacedName)
	ctx = logr.NewContext(ctx, reqLogger)
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

	podRunnerPairs, err := r.getPodRunnerPairs(ctx, instance)
	if err != nil {
		return r.manageOutcome(ctx, instance, err)
	}

	if !podRunnerPairs.inSync() {
		reqLogger.Info("Pods and runner API not in sync, returning early")
		return r.manageOutcome(ctx, instance, nil)
	}

	// if under desired minimum instances or pool is saturated, scale up
	if podRunnerPairs.numRunners() < instance.Spec.MinRunners || (podRunnerPairs.allBusy() && podRunnerPairs.numRunners() < instance.Spec.MaxRunners) {
		instance.Status.CurrentSize = podRunnerPairs.numPods()
		scale := funk.MaxInt([]int{instance.Spec.MinRunners - podRunnerPairs.numRunners(), 1}).(int)
		reqLogger.Info("Scaling up", "numInstances", scale)
		if err := r.scaleUp(ctx, scale, instance); err != nil {
			return r.manageOutcome(ctx, instance, err)
		}
		instance.Status.CurrentSize += scale
		err = r.GetClient().Status().Update(ctx, instance)

		return r.manageOutcome(ctx, instance, err)
	} else if podRunnerPairs.numRunners() > instance.Spec.MaxRunners || ((!podRunnerPairs.allBusy()) && podRunnerPairs.numRunners() > instance.Spec.MinRunners) {
		reqLogger.Info("Scaling down", "totalrunners at github", podRunnerPairs.numRunners(), "maxrunners in CR", instance.Spec.MaxRunners)

		for _, pod := range podRunnerPairs.getIdlePods() {
			err := r.DeleteResourceIfExists(ctx, &pod)
			if err == nil {
				r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Deleted pod %s/%s", pod.Namespace, pod.Name))
				instance.Status.CurrentSize--
				err := r.GetClient().Status().Update(ctx, instance)
				if err != nil {
					return r.manageOutcome(ctx, instance, err)
				}
			}

			return r.manageOutcome(ctx, instance, err)
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
		WithEventFilter(predicate.Funcs{
			// ignore updates to status: https://stuartleeks.com/posts/kubebuilder-event-filters-part-2-update/
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}

func (r *GithubActionRunnerReconciler) scaleUp(ctx context.Context, amount int, instance *garov1alpha1.GithubActionRunner) error {
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
			util.AddFinalizer(pod, finalizer)

			return controllerutil.SetControllerReference(instance, pod, r.GetScheme())
		})
		logr.FromContext(ctx).Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name, "result", result)
		if err != nil {
			return err
		}
		r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Created pod %s/%s", pod.Namespace, pod.Name))
	}

	return nil
}

func (r *GithubActionRunnerReconciler) listRelatedPods(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		//would be safer with owner-ref too, but whatever
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{poolLabel: cr.Name},
	}
	err := r.GetClient().List(ctx, podList, opts...)
	podList.Items = funk.Filter(podList.Items, func(pod corev1.Pod) bool {
		return util.IsOwner(cr, &pod) && !util.IsBeingDeleted(&pod)
	}).([]corev1.Pod)

	return podList, err
}

func (r *GithubActionRunnerReconciler) unregisterRunners(ctx context.Context, cr *garov1alpha1.GithubActionRunner, token string, podRunnerPairs []podRunnerPair) error {
	beingDeleted := funk.Filter(podRunnerPairs, func(pair podRunnerPair) bool {
		return util.IsBeingDeleted(&pair.pod)
	}).([]podRunnerPair)

	for _, item := range beingDeleted {
		if util.HasFinalizer(&item.pod, finalizer) {
			logr.FromContext(ctx).Info("Unregistering runner")
			err := r.GithubAPI.UnregisterRunner(ctx, cr.Spec.Organization, cr.Spec.Repository, token, *item.runner.ID)
			if err != nil {
				return err
			}
			util.RemoveFinalizer(&item.pod, finalizer)
			err = r.GetClient().Update(ctx, &item.pod)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *GithubActionRunnerReconciler) tokenForRef(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (string, error) {
	var secret corev1.Secret
	err := r.GetClient().Get(ctx, client.ObjectKey{Name: cr.Spec.TokenRef.Name, Namespace: cr.Namespace}, &secret)

	if err != nil {
		return "", err
	}
	return string(secret.Data[cr.Spec.TokenRef.Key]), nil
}

func (r *GithubActionRunnerReconciler) getPodRunnerPairs(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (podRunnerPairList, error) {
	var podRunnerPairList podRunnerPairList

	podList, err := r.listRelatedPods(ctx, cr)
	if err != nil {
		return podRunnerPairList, err
	}

	token, err := r.tokenForRef(ctx, cr)
	if err != nil {
		return podRunnerPairList, err
	}

	allRunners, err := r.GithubAPI.GetRunners(cr.Spec.Organization, cr.Spec.Repository, token)
	runners := funk.Filter(allRunners, func(r *github.Runner) bool {
		return strings.HasPrefix(r.GetName(), cr.Name)
	}).([]*github.Runner)

	if err != nil {
		return podRunnerPairList, err
	}

	return from(podList, runners), err
}
