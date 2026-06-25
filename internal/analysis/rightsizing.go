package analysis

import (
	"fmt"
	"math"

	"github.com/pkalab/k8s-cost-optimizer/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type RightSizeResult struct {
	RecommendedCPU    string
	RecommendedMemory string
	SavingsPercent    float64
	Details           string
}

func RightSize(usage v1alpha1.UtilizationStatus) *RightSizeResult {
	cpuP95 := quantityToFloat(usage.CPU.P95Usage)
	memP95 := quantityToFloat(usage.Memory.P95Usage)

	cpuRequested := quantityToFloat(usage.CPU.Requested)
	memRequested := quantityToFloat(usage.Memory.Requested)

	recommendedCPU := ceilToNearest(cpuP95*1.1, 100)
	recommendedMem := ceilToNearest(memP95*1.1, 50)

	if recommendedCPU >= cpuRequested && recommendedMem >= memRequested {
		return nil
	}

	cpuCost := quantityToFloat(usage.ClusterCostPerCPU)
	memCost := quantityToFloat(usage.ClusterCostPerMem)

	currentCost := (cpuRequested * cpuCost) + (memRequested * memCost)
	newCost := (recommendedCPU * cpuCost) + (recommendedMem * memCost)
	savings := 0.0
	if currentCost > 0 {
		savings = ((currentCost - newCost) / currentCost) * 100
	}

	return &RightSizeResult{
		RecommendedCPU:    fmt.Sprintf("%dm", int(recommendedCPU)),
		RecommendedMemory: fmt.Sprintf("%dMi", int(recommendedMem)),
		SavingsPercent:    math.Round(savings*10) / 10,
		Details:           fmt.Sprintf("Requests can be reduced %.0f%% based on p95 utilization", savings),
	}
}

func quantityToFloat(s string) float64 {
	q := resource.MustParse(s)
	return float64(q.MilliValue()) / 1000
}

func ceilToNearest(val float64, nearest float64) float64 {
	return math.Ceil(val/nearest) * nearest
}
