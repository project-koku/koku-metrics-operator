package controller

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
	"github.com/project-koku/koku-metrics-operator/internal/collector"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
)

var (
	now                               = time.Now
	fourteenDayDuration               = time.Duration(14 * 24 * time.Hour)
	ninetyDayDuration                 = time.Duration(90 * 24 * time.Hour)
	retentionPeriod     time.Duration = 0

	monitoringMeta = types.NamespacedName{Namespace: "openshift-monitoring", Name: "cluster-monitoring-config"}

	promCfgSetter  collector.PrometheusConfigurationSetter = collector.SetPrometheusConfig
	promConnSetter collector.PrometheusConnectionSetter    = collector.SetPrometheusConnection
	promConnTester collector.PrometheusConnectionTester    = collector.TestPrometheusConnection

	retryTracker map[time.Time]int = make(map[time.Time]int)
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
	// Here we use k8s APIReader to read the k8s object by making the
	// direct call to k8s apiserver instead of using k8sClient.
	// The reason is that k8sClient uses a cache and we cant populate the cache
	// with openshift-monitoring resources without some custom cache func.
	// It is okay to make direct call to k8s apiserver because we are only
	// making single read call once per CR installation.
	if err := r.apiReader.Get(ctx, monitoringMeta, &configMap); err != nil {
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
	start := now().UTC().Truncate(time.Hour).Add(-time.Hour) // start of previous full hour
	end := start.Add(59*time.Minute + 59*time.Second)

	if retentionPeriod == 0 {
		setRetentionPeriod(ctx, r)
	}

	// logic for gathering previous data upon CR creation
	if cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
		*cr.Spec.PrometheusConfig.CollectPreviousData &&
		cr.Status.Prometheus.LastQuerySuccessTime.IsZero() {
		// LastQuerySuccessTime is zero when the CR is first created. We will only reset `start` to the beginning of
		// the retention period when the CR is first created, otherwise we stick to using the start of the previous full hour.
		log.Info(fmt.Sprintf("duration used: %s", retentionPeriod))
		start = start.Add(-1 * retentionPeriod).Truncate(24 * time.Hour)
		log.Info(fmt.Sprintf("start used: %s", start))
		cr.Status.Prometheus.PreviousDataCollected = true
		r.initialDataCollection = true
		return start, end
	}

	if !cr.Status.Prometheus.LastQuerySuccessTime.IsZero() && start.Sub(cr.Status.Prometheus.LastQuerySuccessTime.Time) > time.Hour {
		// If we have previously had successful queries, and the difference between the last success and the current hour to query
		// is greater than 1 hour, then we've had a query failure. Here, we reset the start of the query hour so that we retry gathering
		// metrics from the last time we had a success
		if start.Add(-1 * retentionPeriod).After(cr.Status.Prometheus.LastQuerySuccessTime.Time) {
			// ensure we aren't trying to gather beyond the retention period
			start = start.Add(-1 * retentionPeriod).Truncate(24 * time.Hour)
		} else {
			start = cr.Status.Prometheus.LastQuerySuccessTime.Add(time.Hour)
		}
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

func isQueryNeeded(start time.Time) bool {
	retryCount, ok := retryTracker[start]
	if !ok {
		// if the time is not in the map, we need to run the queries for this time
		retryTracker[start] = 0
		return true
	}

	if retryCount < 5 {
		// retry up to 5 times
		return true
	}

	// we've exceeded 5 tries, so give up trying
	log.Info("query retry limit exceeded", "start", start.Format(time.RFC3339))
	return false

}

func collectPromStats(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, timeRange promv1.Range) error {
	log := log.WithName("collectPromStats")

	r.promCollector.TimeSeries = &timeRange

	t := metav1.Time{Time: timeRange.Start}
	formattedStart := timeRange.Start.Format(time.RFC3339)
	formattedEnd := timeRange.End.Format(time.RFC3339)
	if cr.Status.Prometheus.LastQuerySuccessTime.Equal(&t) {
		log.Info("reports already generated for range", "start", formattedStart, "end", formattedEnd)
		return nil
	}

	cr.Status.Prometheus.LastQueryStartTime = t

	shouldQuery := isQueryNeeded(timeRange.Start)
	if !shouldQuery {
		// here, we have already tried this timeRange 5 times. The CR contains
		// all the necessary errors at this point, so just return
		return nil
	}

	log.Info("generating reports for range", "start", formattedStart, "end", formattedEnd)
	if err := collector.GenerateReports(cr, dirCfg, r.promCollector); err != nil {
		cr.Status.Reports.DataCollected = false
		if err == collector.ErrNoData {
			cr.Status.Prometheus.LastQuerySuccessTime = t
			cr.Status.Reports.DataCollectionMessage = "No data to report for the hour queried."
			log.Info("no data available to generate reports")
		} else {
			retryTracker[timeRange.Start]++
			cr.Status.Reports.DataCollectionMessage = fmt.Sprintf("error: %v", err)
			log.Error(err, "failed to generate reports")
		}
		return err
	}
	cr.Status.Reports.DataCollected = true
	cr.Status.Reports.DataCollectionMessage = ""
	log.Info("reports generated for range", "start", formattedStart, "end", formattedEnd)
	cr.Status.Prometheus.LastQuerySuccessTime = t

	// since we've had a successful query, we should wipe the tracker to remove it from mem
	retryTracker = make(map[time.Time]int)
	return nil
}
