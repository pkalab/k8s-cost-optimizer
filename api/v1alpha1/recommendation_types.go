package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RecommendationSpec struct {
	WorkloadRef WorkloadRef `json:"workloadRef"`
	Type        string      `json:"type"` // RightSize, SpotInstance, ReservedInstance
}

type ProposedResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type RecommendationStatus struct {
	CurrentCost      float64            `json:"currentCost"`
	RecommendedCost  float64            `json:"recommendedCost"`
	Savings          float64            `json:"savings"`
	SavingsPercent   float64            `json:"savingsPercent"`
	ProposedResources *ProposedResources `json:"proposedResources,omitempty"`
	Applied          bool               `json:"applied"`
	Details          string             `json:"details"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   RecommendationSpec   `json:"spec,omitempty"`
	Status RecommendationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type RecommendationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recommendation `json:"items"`
}

func init() { SchemeBuilder.Register(&Recommendation{}, &RecommendationList{}) }
