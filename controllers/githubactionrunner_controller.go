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
	"strconv"
	"strings"
	"time"
)

const poolLabel = "garo.tietoevry.com/pool"
const finalizer = "garo.tietoevry.com/runner-registration"
const registrationTokenKey = "RUNNER_TOKEN"
const registrationTokenExpiresAtAnnotation = "garo.tietoevry.com/expiryTimestamp"
const regTokenPostfix = "regtoken"

// GithubActionRunnerReconciler reconciles a GithubActionRunner object
type GithubActionRunnerReconciler struct {
	util.ReconcilerBase
	Log       logr.Logger
	GithubAPI githubapi.IRunnerAPI
}

// IsValid validates the CR and returns false if it is not valid along with the validation errors, else true and nil
func (r *GithubActionRunnerReconciler) IsValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*garov1alpha1.GithubActionRunner)
	if !ok {
		return false, errors.New("not a GithubActionRunner object")
	}
	return instance.Spec.IsValid()
}

// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners,verbs="*"
// +kubebuilder:rbac:groups=garo.tietoevry.com,resources=githubactionrunners/*,verbs="*"
// +kubebuilder:rbac:groups=core,resources=pods,verbs="*"
// +kubebuilder:rbac:groups=core,resources=secrets,verbs="*"
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

	return r.handleScaling(ctx, instance)
}

// handleScaling is the main logic of the controller
func (r *GithubActionRunnerReconciler) handleScaling(ctx context.Context, instance *garov1alpha1.GithubActionRunner) (reconcile.Result, error) {
	logger := logr.FromContext(ctx)

	podRunnerPairs, err := r.getPodRunnerPairs(ctx, instance)
	if err != nil {
		return r.manageOutcome(ctx, instance, err)
	}

	// safety guard - always look for finalizers in order to unregister runners for pods about to delete
	if err = r.unregisterRunners(ctx, instance, podRunnerPairs); err != nil {
		return r.manageOutcome(ctx, instance, err)
	}

	if !podRunnerPairs.inSync() {
		logger.Info("Pods and runner API not in sync, returning early")
		return r.manageOutcome(ctx, instance, nil)
	}

	if shouldScaleUp(podRunnerPairs, instance) {
		instance.Status.CurrentSize = podRunnerPairs.numPods()
		scale := funk.MaxInt([]int{instance.Spec.MinRunners - podRunnerPairs.numRunners(), 1}).(int)
		logger.Info("Scaling up", "numInstances", scale)

		if err := r.createOrUpdateRegistrationTokenSecret(ctx, instance); err != nil {
			return r.manageOutcome(ctx, instance, err)
		}

		if err := r.scaleUp(ctx, scale, instance); err != nil {
			return r.manageOutcome(ctx, instance, err)
		}

		instance.Status.CurrentSize += scale
		err = r.GetClient().Status().Update(ctx, instance)

		return r.manageOutcome(ctx, instance, err)
	} else if shouldScaleDown(podRunnerPairs, instance) {
		logger.Info("Scaling down", "runners at github", podRunnerPairs.numRunners(), "maxrunners in CR", instance.Spec.MaxRunners)

		pod := podRunnerPairs.getIdlePods()[0]
		err := r.DeleteResourceIfExists(ctx, &pod)
		if err == nil {
			r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Deleted pod %s/%s", pod.Namespace, pod.Name))
			instance.Status.CurrentSize--
			if err := r.GetClient().Status().Update(ctx, instance); err != nil {
				return r.manageOutcome(ctx, instance, err)
			}
		}

		return r.manageOutcome(ctx, instance, err)
	}

	return r.manageOutcome(ctx, instance, err)
}

func shouldScaleUp(podRunnerPairs podRunnerPairList, instance *garov1alpha1.GithubActionRunner) bool {
	return podRunnerPairs.numRunners() < instance.Spec.MinRunners || (podRunnerPairs.allBusy() && podRunnerPairs.numRunners() < instance.Spec.MaxRunners)
}

func shouldScaleDown(podRunnerPairs podRunnerPairList, instance *garov1alpha1.GithubActionRunner) bool {
	return podRunnerPairs.numRunners() > instance.Spec.MaxRunners || ((!podRunnerPairs.allBusy()) && podRunnerPairs.numRunners() > instance.Spec.MinRunners)
}

func (r *GithubActionRunnerReconciler) manageOutcome(ctx context.Context, instance *garov1alpha1.GithubActionRunner, issue error) (reconcile.Result, error) {
	return r.ManageOutcomeWithRequeue(ctx, instance, issue, instance.Spec.GetReconciliationPeriod())
}

// SetupWithManager configures the controller by using the passed mgr
func (r *GithubActionRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garov1alpha1.GithubActionRunner{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		WithEventFilter(predicate.Funcs{
			// ignore updates to status: https://stuartleeks.com/posts/kubebuilder-event-filters-part-2-update/
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}

func (r *GithubActionRunnerReconciler) createOrUpdateRegistrationTokenSecret(ctx context.Context, instance *garov1alpha1.GithubActionRunner) error {
	logger := logr.FromContext(ctx)
	secret := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
	}
	err := r.GetClient().Get(ctx, client.ObjectKeyFromObject(instance), secret)

	// not found - create
	if apierrors.IsNotFound(err) {
		logger.Info("Registration secret not found, creating")
		return r.updateRegistrationToken(ctx, instance, secret)
	}

	// else a problem - then return the err
	if err != nil {
		return err
	}

	// else found and check validity
	epoch, err := strconv.ParseInt(secret.Annotations[registrationTokenExpiresAtAnnotation], 10, 64)
	if err != nil {
		return err
	}

	expired := time.Unix(epoch, 0).Before(time.Now().Add(-5 * time.Minute))
	if expired {
		logger.Info("Registration token expired, updating")
		return r.updateRegistrationToken(ctx, instance, secret)
	}

	return err
}

func (r *GithubActionRunnerReconciler) updateRegistrationToken(ctx context.Context, instance *garov1alpha1.GithubActionRunner, secret *corev1.Secret) error {
	secret.GetObjectMeta().SetName(fmt.Sprintf("%s-%s", instance.GetName(), regTokenPostfix))
	secret.GetObjectMeta().SetNamespace(instance.GetNamespace())
	apiToken, err := r.tokenForRef(ctx, instance)
	if err != nil {
		return err
	}

	regToken, err := r.GithubAPI.CreateRegistrationToken(ctx, instance.Spec.Organization, instance.Spec.Repository, apiToken)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.GetClient(), secret, func() error {
		objectMeta := secret.GetObjectMeta()
		if err := r.addMetaData(instance, &objectMeta); err != nil {
			return err
		}

		secret.StringData = make(map[string]string)
		secret.StringData[registrationTokenKey] = *regToken.Token
		if secret.GetAnnotations() == nil {
			secret.SetAnnotations(make(map[string]string))
		}
		secret.Annotations[registrationTokenExpiresAtAnnotation] = strconv.FormatInt(regToken.ExpiresAt.Unix(), 10)

		return err
	})

	return err
}

func (r *GithubActionRunnerReconciler) addMetaData(instance *garov1alpha1.GithubActionRunner, object *metav1.Object) error {
	labels := (*object).GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		(*object).SetLabels(labels)
	}
	err := mergo.Merge(&labels, instance.ObjectMeta.Labels)
	if err != nil {
		return err
	}

	labels[poolLabel] = instance.Name

	err = controllerutil.SetControllerReference(instance, *object, r.GetScheme())

	return err
}

func (r *GithubActionRunnerReconciler) scaleUp(ctx context.Context, amount int, instance *garov1alpha1.GithubActionRunner) error {
	for i := 0; i < amount; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-pod-", instance.Name),
				Namespace:    instance.Namespace,
			},
		}
		result, err := controllerutil.CreateOrUpdate(ctx, r.GetClient(), pod, func() error {
			pod.Spec = *instance.Spec.PodTemplateSpec.Spec.DeepCopy()

			meta := pod.GetObjectMeta()
			if err := r.addMetaData(instance, &meta); err != nil {
				return err
			}

			util.AddFinalizer(pod, finalizer)

			return nil
		})
		logr.FromContext(ctx).Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name, "result", result)
		if err != nil {
			return err
		}

		r.GetRecorder().Event(instance, corev1.EventTypeNormal, "Scaling", fmt.Sprintf("Created pod %s/%s", pod.Namespace, pod.Name))
	}

	return nil
}

// listRelatedPods returns pods related to the GithubActionRunner
func (r *GithubActionRunnerReconciler) listRelatedPods(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(cr.Namespace),
		client.MatchingLabels{poolLabel: cr.Name},
	}
	if err := r.GetClient().List(ctx, podList, opts...); err != nil {
		return nil, err
	}

	// filter result by owner-ref since it cannot be done server-side
	podList.Items = funk.Filter(podList.Items, func(pod corev1.Pod) bool {
		return util.IsOwner(cr, &pod)
	}).([]corev1.Pod)

	return podList, nil
}

// unregisterRunners will remove runner from github based on presence of finalizer
func (r *GithubActionRunnerReconciler) unregisterRunners(ctx context.Context, cr *garov1alpha1.GithubActionRunner, list podRunnerPairList) error {
	for _, item := range list.getPodsBeingDeleted() {
		if util.HasFinalizer(&item.pod, finalizer) {
			logr.FromContext(ctx).Info("Unregistering runner", "name", item.runner.GetName(), "id", item.runner.GetID())
			token, err := r.tokenForRef(ctx, cr)
			if err != nil {
				return err
			}
			if err = r.GithubAPI.UnregisterRunner(ctx, cr.Spec.Organization, cr.Spec.Repository, token, *item.runner.ID); err != nil {
				return err
			}
			util.RemoveFinalizer(&item.pod, finalizer)
			if err = r.GetClient().Update(ctx, &item.pod); err != nil {
				return err
			}
		}
	}

	return nil
}

// tokenForRef returns the token referenced from the GithubActionRunner Spec.TokenRef
func (r *GithubActionRunnerReconciler) tokenForRef(ctx context.Context, cr *garov1alpha1.GithubActionRunner) (string, error) {
	var secret corev1.Secret
	if err := r.GetClient().Get(ctx, client.ObjectKey{Name: cr.Spec.TokenRef.Name, Namespace: cr.Namespace}, &secret); err != nil {
		return "", err
	}

	return string(secret.Data[cr.Spec.TokenRef.Key]), nil
}

// getPodRunnerPairs returns a struct podRunnerPairList with pods and runners
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

	allRunners, err := r.GithubAPI.GetRunners(ctx, cr.Spec.Organization, cr.Spec.Repository, token)
	runners := funk.Filter(allRunners, func(r *github.Runner) bool {
		return strings.HasPrefix(r.GetName(), cr.Name)
	}).([]*github.Runner)

	if err != nil {
		return podRunnerPairList, err
	}

	return from(podList, runners), err
}
