package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type CostOptimizerConfigSpec struct {
	LookbackPeriod          string  `json:"lookbackPeriod"`
	PrometheusAddress       string  `json:"prometheusAddress"`
	AWSRegion               string  `json:"awsRegion"`
	SpotCandidateThreshold  float64 `json:"spotCandidateThreshold"`
	RITermYears             int     `json:"riTermYears"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type CostOptimizerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec CostOptimizerConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type CostOptimizerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CostOptimizerConfig `json:"items"`
}

func init() { SchemeBuilder.Register(&CostOptimizerConfig{}, &CostOptimizerConfigList{}) }
