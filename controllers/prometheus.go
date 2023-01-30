package controllers

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/collector"
)

// reconcilePrometheusRule reconciles the PrometheusRule
func (r *KokuMetricsConfigReconciler) reconcilePrometheusRule(cr *kokumetricscfgv1beta1.KokuMetricsConfig) error {

	promRule := newPrometheusRule(cr.Namespace, "koku-metrics-promrule")

	if err := r.Get(context.Background(), types.NamespacedName{Name: promRule.Name, Namespace: promRule.Namespace}, promRule); err == nil {

		// if !cr.Spec.Monitoring.Enabled {
		// 	// PrometheusRule exists but enabled flag has been set to false, delete the PrometheusRule
		// 	log.Info("instance monitoring disabled, deleting component status tracking prometheusRule")
		// 	return r.Client.Delete(context.TODO(), promRule)
		// }
		return nil // PrometheusRule found, do nothing
	}

	ruleGroups := []monitoringv1.RuleGroup{
		{
			Name: "cost_management:node",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:cost:node_allocatable_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:node_allocatable_cpu_cores"],
					},
				},
				{
					Record: "koku_metrics:cost:node_capacity_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:node_capacity_cpu_cores"],
					},
				},
				{
					Record: "koku_metrics:cost:node_allocatable_memory_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:node_allocatable_memory_bytes"],
					},
				},
				{
					Record: "koku_metrics:cost:node_capacity_memory_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:node_capacity_memory_bytes"],
					},
				},
			},
		},
		{
			Name: "cost_management:volume",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:cost:persistentvolume_pod_info",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:persistentvolume_pod_info"],
					},
				},
				{
					Record: "koku_metrics:cost:persistentvolumeclaim_capacity_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:persistentvolumeclaim_capacity_bytes"],
					},
				},
				{
					Record: "koku_metrics:cost:persistentvolumeclaim_request_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:persistentvolumeclaim_request_bytes"],
					},
				},
				{
					Record: "koku_metrics:cost:persistentvolumeclaim_usage_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:persistentvolumeclaim_usage_bytes"],
					},
				},
			},
		},
		{
			Name: "cost_management:pod",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:cost:pod_limit_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_limit_cpu_cores"],
					},
				},
				{
					Record: "koku_metrics:cost:pod_request_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_request_cpu_cores"],
					},
				},
				{
					Record: "koku_metrics:cost:pod_usage_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_usage_cpu_cores"],
					},
				},
				{
					Record: "koku_metrics:cost:pod_limit_memory_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_limit_memory_bytes"],
					},
				},
				{
					Record: "koku_metrics:cost:pod_request_memory_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_request_memory_bytes"],
					},
				},
				{
					Record: "koku_metrics:cost:pod_usage_memory_bytes",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:cost:pod_usage_memory_bytes"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:cpu_request_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:cpu_request_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_request_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_request_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_request_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:cpu_limit_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:cpu_limit_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_limit_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_limit_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_limit_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:cpu_usage_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:cpu_usage_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_usage_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_usage_container_min"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_usage_container_max"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_usage_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:cpu_throttle_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:cpu_throttle_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_throttle_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_throttle_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_throttle_container_max"],
					},
				},
				{
					Record: "koku_metrics:ros:cpu_throttle_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:cpu_throttle_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:memory_request_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:memory_request_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_request_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_request_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_request_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:memory_limit_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:memory_limit_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_limit_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_limit_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_limit_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:memory_usage_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:memory_usage_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_usage_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_usage_container_min"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_usage_container_max"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_usage_container_sum"],
					},
				},
			},
		},
		{
			Name: "resource_optimization:memory_rss_usage_container",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_avg",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_rss_usage_container_avg"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_rss_usage_container_min"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_rss_usage_container_max"],
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: collector.QueryMap["koku_metrics:ros:memory_rss_usage_container_sum"],
					},
				},
			},
		},
	}
	promRule.Spec.Groups = ruleGroups

	if err := controllerutil.SetControllerReference(cr, promRule, r.Scheme); err != nil {
		return err
	}

	log.Info("instance monitoring enabled, creating component status tracking prometheusRule")
	return r.Client.Create(context.Background(), promRule) // Create PrometheusRule
}

// newPrometheusRule returns the expected PrometheusRule
func newPrometheusRule(namespace, promRuleName string) *monitoringv1.PrometheusRule {

	promRule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promRuleName,
			Namespace: namespace,
		},
		Spec: monitoringv1.PrometheusRuleSpec{},
	}
	return promRule
}
