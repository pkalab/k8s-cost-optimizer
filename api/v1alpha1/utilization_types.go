package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceUsage struct {
	Requested string `json:"requested"`
	AvgUsage  string `json:"avgUsage"`
	P95Usage  string `json:"p95Usage"`
	MaxUsage  string `json:"maxUsage"`
}

type WorkloadRef struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type UtilizationSpec struct {
	WorkloadRef WorkloadRef `json:"workloadRef"`
	Lookback    string      `json:"lookback"`
}

type UtilizationStatus struct {
	CPU              ResourceUsage `json:"cpu"`
	Memory           ResourceUsage `json:"memory"`
	ClusterCostPerCPU string       `json:"clusterCostPerCPU"`
	ClusterCostPerMem string       `json:"clusterCostPerMem"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Utilization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   UtilizationSpec   `json:"spec,omitempty"`
	Status UtilizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type UtilizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Utilization `json:"items"`
}

func init() { SchemeBuilder.Register(&Utilization{}, &UtilizationList{}) }
