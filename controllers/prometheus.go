package controllers

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
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
			Name: "KokuMetricsPrometheusRecords",
			Rules: []monitoringv1.Rule{
				{
					Record: "koku_metrics:cost:node_allocatable_cpu_cores",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "kube_node_status_allocatable{resource='cpu'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
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
						StrVal: "avg by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_request_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
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
						StrVal: "avg by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_limit_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
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
						StrVal: "avg by(container, pod, namespace) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "min by(container, pod, namespace) (min_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "max by(container, pod, namespace) (max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
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
						StrVal: "avg by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_throttle_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "max by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:cpu_throttle_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
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
						StrVal: "avg by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_request_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
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
						StrVal: "avg by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_limit_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
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
						StrVal: "avg by(container, pod, namespace) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "min by(container, pod, namespace) (min_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "max by(container, pod, namespace) (max_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
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
						StrVal: "avg by(container, pod, namespace) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_min",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "min by(container, pod, namespace) (min_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_max",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "max by(container, pod, namespace) (max_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
					},
				},
				{
					Record: "koku_metrics:ros:memory_rss_usage_container_sum",
					Expr: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "sum by(container, pod, namespace) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
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
