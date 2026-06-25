package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	costv1alpha1 "github.com/pkalab/k8s-cost-optimizer/api/v1alpha1"
)

const (
	AnnotationCPU       = "cost-optimizer.io/recommended-cpu"
	AnnotationMem       = "cost-optimizer.io/recommended-mem"
	AnnotationSavings   = "cost-optimizer.io/savings-monthly"
	AnnotationAnalyzed  = "cost-optimizer.io/analyzed-at"
)

type AnnotatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=recommendations,verbs=get;list;watch
// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=recommendations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;patch

func (r *AnnotatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Recommendation")

	var rec costv1alpha1.Recommendation
	if err := r.Get(ctx, req.NamespacedName, &rec); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if rec.Status.Applied {
		return ctrl.Result{}, nil
	}

	nn := types.NamespacedName{
		Name:      rec.Spec.WorkloadRef.Name,
		Namespace: rec.Spec.WorkloadRef.Namespace,
	}

	switch rec.Spec.WorkloadRef.Kind {
	case "Deployment":
		var dep appsv1.Deployment
		if err := r.Get(ctx, nn, &dep); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if dep.Annotations == nil {
			dep.Annotations = make(map[string]string)
		}
		dep.Annotations[AnnotationCPU] = rec.Status.ProposedResources.CPU
		dep.Annotations[AnnotationMem] = rec.Status.ProposedResources.Memory
		dep.Annotations[AnnotationSavings] = fmt.Sprintf("$%.2f", rec.Status.Savings)
		dep.Annotations[AnnotationAnalyzed] = rec.CreationTimestamp.Format("2006-01-02T15:04:05Z")
		if err := r.Update(ctx, &dep); err != nil {
			return ctrl.Result{}, err
		}
	case "StatefulSet":
		var sts appsv1.StatefulSet
		if err := r.Get(ctx, nn, &sts); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if sts.Annotations == nil {
			sts.Annotations = make(map[string]string)
		}
		sts.Annotations[AnnotationCPU] = rec.Status.ProposedResources.CPU
		sts.Annotations[AnnotationMem] = rec.Status.ProposedResources.Memory
		sts.Annotations[AnnotationSavings] = fmt.Sprintf("$%.2f", rec.Status.Savings)
		sts.Annotations[AnnotationAnalyzed] = rec.CreationTimestamp.Format("2006-01-02T15:04:05Z")
		if err := r.Update(ctx, &sts); err != nil {
			return ctrl.Result{}, err
		}
	}

	rec.Status.Applied = true
	if err := r.Status().Update(ctx, &rec); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AnnotatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costv1alpha1.Recommendation{}).
		Complete(r)
}
