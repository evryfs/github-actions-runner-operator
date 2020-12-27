package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GithubActionRunnerSpec defines the desired state of GithubActionRunner
type GithubActionRunnerSpec struct {
	// Your GitHub organization
	// +kubebuilder:validation:Required
	Organization string `json:"organization"`

	// Optional Github repository name, if repo scoped.
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// Minimum pool-size. Note that you need one runner in order for jobs to be schedulable, else they fail claiming no runners match the selector labels.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default=1
	MinRunners int `json:"minRunners"`

	// Maximum pool-size. Must be greater or equal to minRunners
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxRunners int `json:"maxRunners"`

	// +kubebuilder:validation:Required
	PodTemplateSpec v1.PodTemplateSpec `json:"podTemplateSpec"`

	// +kubebuilder:validation:Required
	TokenRef v1.SecretKeySelector `json:"tokenRef"`

	// How often to reconcile/check the runner pool. If undefined the controller uses a default of 1m
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	ReconciliationPeriod string `json:"reconciliationPeriod"`

	// +kubebuilder:validation:Optional
	Valid bool `json:"valid"`
	// +kubebuilder:validation:Optional
	Error bool `json:"error"`
}

// GetReconciliationPeriod returns period as a Duration
func (r GithubActionRunnerSpec) GetReconciliationPeriod() time.Duration {
	duration, err := time.ParseDuration(r.ReconciliationPeriod)
	if err != nil {
		return time.Minute
	}

	return duration
}

// GithubActionRunnerStatus defines the observed state of GithubActionRunner
type GithubActionRunnerStatus struct {
	// the current size of the build pool
	CurrentSize int `json:"currentSize"`
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// GithubActionRunner is the Schema for the githubactionrunners API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=githubactionrunners,scope=Namespaced
// +kubebuilder:printcolumn:name="currentPoolSize",type=integer,JSONPath=`.status.currentSize`
type GithubActionRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubActionRunnerSpec   `json:"spec,omitempty"`
	Status GithubActionRunnerStatus `json:"status,omitempty"`
}

func (m *GithubActionRunner) GetConditions() []metav1.Condition {
	return m.Status.Conditions
}

func (m *GithubActionRunner) SetConditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// GithubActionRunnerList contains a list of GithubActionRunner
type GithubActionRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubActionRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubActionRunner{}, &GithubActionRunnerList{})
}
