package controllers

// import (
// 	"context"

// 	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
// 	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"
// 	"k8s.io/apimachinery/pkg/util/intstr"
// 	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
// )

// // reconcilePrometheusRule reconciles the PrometheusRule
// func (r *KokuMetricsConfigReconciler) reconcilePrometheusRule(cr *kokumetricscfgv1beta1.KokuMetricsConfig) error {

// 	promRule := newPrometheusRule(cr.Namespace, "koku-metrics-promrule")

// 	if err := r.Get(context.Background(), types.NamespacedName{Name: promRule.Name, Namespace: promRule.Namespace}, promRule); err == nil {

// 		// if !cr.Spec.Monitoring.Enabled {
// 		// 	// PrometheusRule exists but enabled flag has been set to false, delete the PrometheusRule
// 		// 	log.Info("instance monitoring disabled, deleting component status tracking prometheusRule")
// 		// 	return r.Client.Delete(context.TODO(), promRule)
// 		// }
// 		return nil // PrometheusRule found, do nothing
// 	}

// 	ruleGroups := []monitoringv1.RuleGroup{
// 		{
// 			Name: "KokuMetricsPrometheusRecords",
// 			Rules: []monitoringv1.Rule{
// 				{
// 					Record: "koku_metrics:node_allocatable_cpu_cores",
// 					Expr: intstr.IntOrString{
// 						Type:   intstr.String,
// 						StrVal: "kube_node_status_allocatable{resource='cpu'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	promRule.Spec.Groups = ruleGroups

// 	if err := controllerutil.SetControllerReference(cr, promRule, r.Scheme); err != nil {
// 		return err
// 	}

// 	log.Info("instance monitoring enabled, creating component status tracking prometheusRule")
// 	return r.Client.Create(context.Background(), promRule) // Create PrometheusRule
// }

// // newPrometheusRule returns the expected PrometheusRule
// func newPrometheusRule(namespace, promRuleName string) *monitoringv1.PrometheusRule {

// 	promRule := &monitoringv1.PrometheusRule{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      promRuleName,
// 			Namespace: namespace,
// 		},
// 		Spec: monitoringv1.PrometheusRuleSpec{},
// 	}
// 	return promRule
// }
