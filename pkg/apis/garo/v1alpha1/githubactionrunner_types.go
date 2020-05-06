package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GithubActionRunnerSpec defines the desired state of GithubActionRunner
type GithubActionRunnerSpec struct {
	Organization string `json:"organization"`
	// +kubebuilder:validation:Minimum=0
	MinRunners int `json:"minRunners"`
	// +kubebuilder:validation:Minimum=0
	MaxRunners int                  `json:"maxRunners"`
	PodSpec    v1.PodSpec           `json:"podSpec"`
	TokenRef   v1.SecretKeySelector `json:"tokenRef"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// GithubActionRunnerStatus defines the observed state of GithubActionRunner
type GithubActionRunnerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GithubActionRunner is the Schema for the githubactionrunners API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=githubactionrunners,scope=Namespaced
type GithubActionRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubActionRunnerSpec   `json:"spec,omitempty"`
	Status GithubActionRunnerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GithubActionRunnerList contains a list of GithubActionRunner
type GithubActionRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubActionRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubActionRunner{}, &GithubActionRunnerList{})
}
