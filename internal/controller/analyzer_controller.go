package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	costv1alpha1 "github.com/pkalab/k8s-cost-optimizer/api/v1alpha1"
	"github.com/pkalab/k8s-cost-optimizer/internal/analysis"
)

type AnalyzerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=utilizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=recommendations,verbs=get;list;watch;create;update;patch

func (r *AnalyzerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Utilization")

	var util costv1alpha1.Utilization
	if err := r.Get(ctx, req.NamespacedName, &util); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	rightSize := analysis.RightSize(util.Status)
	if rightSize == nil {
		return ctrl.Result{}, nil
	}

	rec := &costv1alpha1.Recommendation{
		Spec: costv1alpha1.RecommendationSpec{
			WorkloadRef: util.Spec.WorkloadRef,
			Type:        "RightSize",
		},
	}
	rec.Name = fmt.Sprintf("%s-%s-rightsize", util.Spec.WorkloadRef.Name, util.Spec.WorkloadRef.Namespace)
	rec.Namespace = util.Spec.WorkloadRef.Namespace
	rec.Status = costv1alpha1.RecommendationStatus{
		CurrentCost:     100.0,
		RecommendedCost: 100.0 * (1 - rightSize.SavingsPercent/100),
		Savings:         100.0 * rightSize.SavingsPercent / 100,
		SavingsPercent:  rightSize.SavingsPercent,
		ProposedResources: &costv1alpha1.ProposedResources{
			CPU:    rightSize.RecommendedCPU,
			Memory: rightSize.RecommendedMemory,
		},
		Applied: false,
		Details: rightSize.Details,
	}

	if err := r.Patch(ctx, rec, client.Apply, client.ForceOwnership, client.FieldOwner("analyzer")); err != nil {
		logger.Error(err, "failed to apply Recommendation")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AnalyzerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costv1alpha1.Utilization{}).
		Complete(r)
}
