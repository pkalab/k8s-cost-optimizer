package analysis

import (
	"testing"

	"github.com/pkalab/k8s-cost-optimizer/api/v1alpha1"
)

func TestRightSize_Reduction(t *testing.T) {
	status := v1alpha1.UtilizationStatus{
		CPU:              v1alpha1.ResourceUsage{Requested: "1000m", P95Usage: "500m"},
		Memory:           v1alpha1.ResourceUsage{Requested: "1024Mi", P95Usage: "256Mi"},
		ClusterCostPerCPU: "0.00003",
		ClusterCostPerMem: "0.000004",
	}
	result := RightSize(status)
	if result == nil {
		t.Fatal("expected recommendation, got nil")
	}
}

func TestRightSize_NoReduction(t *testing.T) {
	status := v1alpha1.UtilizationStatus{
		CPU:    v1alpha1.ResourceUsage{Requested: "500m", P95Usage: "900m"},
		Memory: v1alpha1.ResourceUsage{Requested: "256Mi", P95Usage: "512Mi"},
	}
	result := RightSize(status)
	if result != nil {
		t.Fatal("expected nil (no reduction possible), got recommendation")
	}
}
