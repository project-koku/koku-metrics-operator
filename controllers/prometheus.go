package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/collector"
	"github.com/project-koku/koku-metrics-operator/dirconfig"
)

var (
	fourteenDayDuration = time.Duration(14 * 24 * time.Hour)
	ninetyDayDuration   = time.Duration(90 * 24 * time.Hour)
	retentionPeriod     time.Duration

	promCompareFormat = "2006-01-02T15"

	monitoringMeta = types.NamespacedName{Namespace: "openshift-monitoring", Name: "cluster-monitoring-config"}

	promCfgSetter  collector.PrometheusConfigurationSetter = collector.SetPrometheusConfig
	promConnSetter collector.PrometheusConnectionSetter    = collector.SetPrometheusConnection
	promConnTester collector.PrometheusConnectionTester    = collector.TestPrometheusConnection
)

type PrometheusK8s struct {
	Retention string `yaml:"retention"`
}

type MonitoringConfig struct {
	PrometheusK8s PrometheusK8s `yaml:"prometheusK8s"`
}

func setRetentionPeriod(ctx context.Context, r *MetricsConfigReconciler) {
	retentionPeriod = fourteenDayDuration

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, monitoringMeta, &configMap); err != nil {
		log.Info(fmt.Sprintf("monitoring configMap not found. defaulting retention to: %s", retentionPeriod))
		return
	}
	data, ok := configMap.Data["config.yaml"]
	if !ok {
		log.Info(fmt.Sprintf("`config.yaml` not found: using default %s", retentionPeriod))
		return
	}

	var mc MonitoringConfig
	if err := yaml.Unmarshal([]byte(data), &mc); err != nil {
		log.Info(fmt.Sprintf("error unmarshalling monitoring config: %v: using default %s", err, retentionPeriod))
		return
	}
	if mc.PrometheusK8s.Retention == "" {
		log.Info(fmt.Sprintf("no retention period defined: using default %s", retentionPeriod))
		return
	}
	timeDuration, err := model.ParseDuration(mc.PrometheusK8s.Retention)
	if err != nil {
		log.Info(fmt.Sprintf("error parsing retention time: %v: using default %s", err, retentionPeriod))
		return
	}
	retentionPeriod = time.Duration(timeDuration)
	if retentionPeriod > ninetyDayDuration {
		retentionPeriod = ninetyDayDuration
		log.Info(fmt.Sprintf("configured retention period greater than 90d: setting to 90d collection: %s", retentionPeriod))
	}
}

func getTimeRange(ctx context.Context, r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) (time.Time, time.Time) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour) // start of previous full hour
	end := start.Add(59*time.Minute + 59*time.Second)
	if cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
		*cr.Spec.PrometheusConfig.CollectPreviousData &&
		cr.Status.Prometheus.LastQuerySuccessTime.IsZero() &&
		!r.disablePreviousDataCollection {
		// LastQuerySuccessTime is zero when the CR is first created. We will only reset `start` to the first of the
		// month when the CR is first created, otherwise we stick to using the start of the previous full hour.
		setRetentionPeriod(ctx, r)
		log.Info(fmt.Sprintf("duration used: %s", retentionPeriod))
		start = start.Add(-1 * retentionPeriod)
		log.Info(fmt.Sprintf("start used: %s", start))
		cr.Status.Prometheus.PreviousDataCollected = true
	}
	return start, end
}

func getPromCollector(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) error {
	if r.promCollector == nil {
		var serviceaccountPath string
		if r.overrideSecretPath {
			val, ok := os.LookupEnv("SECRET_ABSPATH")
			if ok {
				serviceaccountPath = val
			}
		}
		r.promCollector = collector.NewPromCollector(serviceaccountPath)
	}
	r.promCollector.TimeSeries = nil
	if cr.Spec.PrometheusConfig.ContextTimeout == nil {
		timeout := metricscfgv1beta1.DefaultPrometheusContextTimeout
		cr.Spec.PrometheusConfig.ContextTimeout = &timeout
	}
	r.promCollector.ContextTimeout = time.Duration(*cr.Spec.PrometheusConfig.ContextTimeout * int64(time.Second))

	return r.promCollector.GetPromConn(cr, promCfgSetter, promConnSetter, promConnTester)
}

func collectPromStats(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, timeRange promv1.Range) {
	log := log.WithName("collectPromStats")

	r.promCollector.TimeSeries = &timeRange

	t := metav1.Time{Time: timeRange.Start}
	formattedStart := timeRange.Start.Format(time.RFC3339)
	formattedEnd := timeRange.End.Format(time.RFC3339)
	if cr.Status.Prometheus.LastQuerySuccessTime.UTC().Format(promCompareFormat) == t.Format(promCompareFormat) {
		log.Info("reports already generated for range", "start", formattedStart, "end", formattedEnd)
		return
	}

	cr.Status.Prometheus.LastQueryStartTime = t

	log.Info("generating reports for range", "start", formattedStart, "end", formattedEnd)
	if err := collector.GenerateReports(cr, dirCfg, r.promCollector); err != nil {
		cr.Status.Reports.DataCollected = false
		cr.Status.Reports.DataCollectionMessage = fmt.Sprintf("error: %v", err)
		log.Error(err, "failed to generate reports")
		return
	}
	log.Info("reports generated for range", "start", formattedStart, "end", formattedEnd)
	cr.Status.Prometheus.LastQuerySuccessTime = t
}
