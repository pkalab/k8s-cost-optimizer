/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	costv1alpha1 "github.com/pkalab/k8s-cost-optimizer/api/v1alpha1"
	"github.com/pkalab/k8s-cost-optimizer/internal/pricing"
	prom "github.com/pkalab/k8s-cost-optimizer/internal/prometheus"
)

var safeLabelValue = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type CollectorReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	PromClient *prom.Client
	Pricing    *pricing.Client
}

// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=costoptimizerconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=cost.cost-optimizer.io,resources=utilizations,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch

func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling CostOptimizerConfig")

	var config costv1alpha1.CostOptimizerConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	lookback, err := time.ParseDuration(config.Spec.LookbackPeriod)
	if err != nil {
		logger.Error(err, "invalid lookbackPeriod, falling back to 24h", "value", config.Spec.LookbackPeriod)
		lookback = 24 * time.Hour
	}
	end := time.Now()
	start := end.Add(-lookback)

	clusterCost, err := r.Pricing.GetClusterCost(ctx, start, end)
	if err != nil {
		logger.Error(err, "failed to get cluster cost")
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	var deployments corev1.PodList
	if err := r.List(ctx, &deployments); err != nil {
		return ctrl.Result{}, err
	}

	for _, pod := range deployments.Items {
		if len(pod.OwnerReferences) == 0 {
			continue
		}
		owner := pod.OwnerReferences[0]
		if owner.Kind != "Deployment" && owner.Kind != "StatefulSet" {
			continue
		}

		if !safeLabelValue.MatchString(pod.Namespace) || !safeLabelValue.MatchString(pod.Name) {
			logger.Info("skipping pod with invalid label values", "namespace", pod.Namespace, "name", pod.Name)
			continue
		}

		cpuQuery := fmt.Sprintf(
			`avg(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s"}[5m])) by (pod)`,
			pod.Namespace, pod.Name,
		)
		memQuery := fmt.Sprintf(
			`avg(container_memory_working_set_bytes{namespace="%s",pod="%s"}) by (pod)`,
			pod.Namespace, pod.Name,
		)

		cpuSamples, err := r.PromClient.QueryRange(ctx, cpuQuery, start, end, time.Hour)
		if err != nil {
			logger.Error(err, "prometheus CPU query failed", "pod", pod.Name)
			continue
		}
		memSamples, err := r.PromClient.QueryRange(ctx, memQuery, start, end, time.Hour)
		if err != nil {
			logger.Error(err, "prometheus memory query failed", "pod", pod.Name)
			continue
		}

		util := &costv1alpha1.Utilization{
			Spec: costv1alpha1.UtilizationSpec{
				WorkloadRef: costv1alpha1.WorkloadRef{
					Kind:      owner.Kind,
					Name:      owner.Name,
					Namespace: pod.Namespace,
				},
				Lookback: config.Spec.LookbackPeriod,
			},
		}
		util.Name = fmt.Sprintf("%s-%s", owner.Name, pod.Namespace)

		util.Status = buildUtilizationStatus(cpuSamples, memSamples, clusterCost)

		if err := r.Patch(ctx, util, client.Apply, client.ForceOwnership, client.FieldOwner("collector")); err != nil {
			logger.Error(err, "failed to apply Utilization", "pod", pod.Name)
			continue
		}
	}

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

func buildUtilizationStatus(cpuSamples, memSamples []prom.Sample, cost *pricing.ClusterCost) costv1alpha1.UtilizationStatus {
	cpuVals := make([]float64, len(cpuSamples))
	for i, s := range cpuSamples {
		cpuVals[i] = s.Value
	}
	memVals := make([]float64, len(memSamples))
	for i, s := range memSamples {
		memVals[i] = s.Value
	}

	return costv1alpha1.UtilizationStatus{
		CPU:               computeStats(cpuVals, "1000m"),
		Memory:            computeStats(memVals, "512Mi"),
		ClusterCostPerCPU: fmt.Sprintf("%.8f", cost.CPUPerHour),
		ClusterCostPerMem: fmt.Sprintf("%.8f", cost.MemPerGiBHour),
	}
}

func computeStats(vals []float64, requested string) costv1alpha1.ResourceUsage {
	if len(vals) == 0 {
		return costv1alpha1.ResourceUsage{Requested: requested}
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	avg := sum / float64(len(vals))

	max := vals[0]
	for _, v := range vals {
		if v > max {
			max = v
		}
	}

	u := make([]float64, len(vals))
	copy(u, vals)
	sort.Float64s(u)
	idx := int(math.Ceil(float64(len(u))*0.95) - 1)
	if idx < 0 {
		idx = 0
	}
	p95 := u[idx]

	return costv1alpha1.ResourceUsage{
		Requested: requested,
		AvgUsage:  fmt.Sprintf("%dm", int(avg*1000)),
		P95Usage:  fmt.Sprintf("%dm", int(p95*1000)),
		MaxUsage:  fmt.Sprintf("%dm", int(max*1000)),
	}
}

func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costv1alpha1.CostOptimizerConfig{}).
		Complete(r)
}
