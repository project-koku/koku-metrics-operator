//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"

	gologr "github.com/go-logr/logr"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	cv "github.com/project-koku/koku-metrics-operator/clusterversion"
	"github.com/project-koku/koku-metrics-operator/collector"
	"github.com/project-koku/koku-metrics-operator/crhchttp"
	"github.com/project-koku/koku-metrics-operator/dirconfig"
	"github.com/project-koku/koku-metrics-operator/packaging"
	"github.com/project-koku/koku-metrics-operator/sources"
	"github.com/project-koku/koku-metrics-operator/storage"
)

var (
	GitCommit string

	openShiftConfigNamespace = "openshift-config"
	pullSecretName           = "pull-secret"
	pullSecretDataKey        = ".dockerconfigjson"
	pullSecretAuthKey        = "cloud.openshift.com"
	authSecretUserKey        = "username"
	authSecretPasswordKey    = "password"

	falseDef = false
	trueDef  = true

	dirCfg             *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	sourceSpec         *metricscfgv1beta1.CloudDotRedHatSourceSpec
	previousValidation *previousAuthValidation

	log = logr.Log.WithName("metricsconfig_controller")
)

// MetricsConfigReconciler reconciles a MetricsConfig object
type MetricsConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	InCluster bool
	Namespace string

	promCollector                 *collector.PrometheusCollector
	disablePreviousDataCollection bool
	overrideSecretPath            bool
}

// GetClientset returns a clientset based on rest.config
func GetClientset() (*kubernetes.Clientset, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// stringReflectSpec Determine if the string Status item reflects the Spec item if not empty, otherwise take the default value.
func stringReflectSpec(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, specItem *string, statusItem *string, defaultVal string) (string, bool) {
	// Update statusItem if needed
	changed := false
	if *statusItem == "" || !reflect.DeepEqual(*specItem, *statusItem) {
		// If data is specified in the spec it should be used
		changed = true
		if *specItem != "" {
			*statusItem = *specItem
		} else if defaultVal != "" {
			*statusItem = defaultVal
		} else {
			*statusItem = *specItem
		}
	}
	return *statusItem, changed
}

// reflectSpec Determine if the Status item reflects the Spec item if not empty, otherwise set a default value if applicable.
func reflectSpec(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) {

	stringReflectSpec(r, cr, &cr.Spec.APIURL, &cr.Status.APIURL, metricscfgv1beta1.DefaultAPIURL)
	stringReflectSpec(r, cr, &cr.Spec.Authentication.AuthenticationSecretName, &cr.Status.Authentication.AuthenticationSecretName, "")

	if !reflect.DeepEqual(cr.Spec.Authentication.AuthType, cr.Status.Authentication.AuthType) {
		cr.Status.Authentication.AuthType = cr.Spec.Authentication.AuthType
	}
	cr.Status.Upload.ValidateCert = cr.Spec.Upload.ValidateCert

	stringReflectSpec(r, cr, &cr.Spec.Upload.IngressAPIPath, &cr.Status.Upload.IngressAPIPath, metricscfgv1beta1.DefaultIngressPath)
	cr.Status.Upload.UploadToggle = cr.Spec.Upload.UploadToggle

	// set the default max file size for packaging
	cr.Status.Packaging.MaxSize = &cr.Spec.Packaging.MaxSize
	cr.Status.Packaging.MaxReports = &cr.Spec.Packaging.MaxReports

	// set the upload wait to whatever is in the spec, if the spec is defined
	if cr.Spec.Upload.UploadWait != nil {
		cr.Status.Upload.UploadWait = cr.Spec.Upload.UploadWait
	}

	// if the status is nil, generate an upload wait
	if cr.Status.Upload.UploadWait == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		uploadWait := r.Int63() % 35
		cr.Status.Upload.UploadWait = &uploadWait
	}

	if !reflect.DeepEqual(cr.Spec.Upload.UploadCycle, cr.Status.Upload.UploadCycle) {
		cr.Status.Upload.UploadCycle = cr.Spec.Upload.UploadCycle
	}

	stringReflectSpec(r, cr, &cr.Spec.Source.SourcesAPIPath, &cr.Status.Source.SourcesAPIPath, metricscfgv1beta1.DefaultSourcesPath)
	stringReflectSpec(r, cr, &cr.Spec.Source.SourceName, &cr.Status.Source.SourceName, "")

	cr.Status.Source.CreateSource = cr.Spec.Source.CreateSource

	if !reflect.DeepEqual(cr.Spec.Source.CheckCycle, cr.Status.Source.CheckCycle) {
		cr.Status.Source.CheckCycle = cr.Spec.Source.CheckCycle
	}

	stringReflectSpec(r, cr, &cr.Spec.PrometheusConfig.SvcAddress, &cr.Status.Prometheus.SvcAddress, metricscfgv1beta1.DefaultPrometheusSvcAddress)
	cr.Status.Prometheus.SkipTLSVerification = cr.Spec.PrometheusConfig.SkipTLSVerification
	cr.Status.Prometheus.ContextTimeout = cr.Spec.PrometheusConfig.ContextTimeout
	cr.Status.Prometheus.DisabledMetricsCollectionCostManagement = cr.Spec.PrometheusConfig.DisableMetricsCollectionCostManagement
	cr.Status.Prometheus.DisabledMetricsCollectionResourceOptimization = cr.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization
}

func setClusterID(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) error {
	if cr.Status.ClusterID != "" && cr.Status.ClusterVersion != "" {
		return nil
	}

	cvClient := cv.NewCVClient(r.Client)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}

	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		cr.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
	}
	if clusterVersion.Status.Desired.Version != "" {
		cr.Status.ClusterVersion = string(clusterVersion.Status.Desired.Version)
	}
	return nil
}

func setOperatorCommit(r *MetricsConfigReconciler) {
	log := log.WithName("setOperatorCommit")
	if GitCommit == "" {
		commit, exists := os.LookupEnv("GIT_COMMIT")
		if exists {
			msg := fmt.Sprintf("using git commit from environment: %s", commit)
			log.Info(msg)
			GitCommit = commit
		}
	}
}

func checkCycle(log gologr.Logger, cycle int64, lastExecution metav1.Time, action string) bool {
	log = log.WithName("checkCycle")

	if lastExecution.IsZero() {
		log.Info(fmt.Sprintf("there have been no prior successful %ss", action))
		return true
	}

	duration := time.Since(lastExecution.Time.UTC())
	minutes := int64(duration.Minutes())
	log.Info(fmt.Sprintf("it has been %d minute(s) since the last successful %s", minutes, action))
	if minutes >= cycle {
		log.Info(fmt.Sprintf("executing %s", action))
		return true
	}
	log.Info(fmt.Sprintf("not time to execute the %s", action))
	return false

}

func checkSource(r *MetricsConfigReconciler, handler *sources.SourceHandler, cr *metricscfgv1beta1.MetricsConfig) {
	log := log.WithName("checkSource")

	// check if the Source Spec has changed
	updated := false
	if sourceSpec != nil {
		updated = !reflect.DeepEqual(*sourceSpec, cr.Spec.Source)
	}
	sourceSpec = cr.Spec.Source.DeepCopy()

	if handler.Spec.SourceName != "" && (updated || checkCycle(log, *handler.Spec.CheckCycle, handler.Spec.LastSourceCheckTime, "source check")) {
		client := crhchttp.GetClient(handler.Auth)
		cr.Status.Source.SourceError = ""
		defined, lastCheck, err := sources.SourceGetOrCreate(handler, client)
		if err != nil {
			cr.Status.Source.SourceError = err.Error()
			log.Info("source get or create message", "error", err)
		}
		cr.Status.Source.SourceDefined = &defined
		cr.Status.Source.LastSourceCheckTime = lastCheck
	}
}

func configurePVC(r *MetricsConfigReconciler, req ctrl.Request, cr *metricscfgv1beta1.MetricsConfig) (*ctrl.Result, error) {
	ctx := context.Background()
	log := log.WithName("configurePVC")
	pvcTemplate := cr.Spec.VolumeClaimTemplate
	if pvcTemplate == nil {
		pvcTemplate = &storage.DefaultPVC
	}

	stor := &storage.Storage{
		Client:    r.Client,
		CR:        cr,
		Namespace: req.Namespace,
		PVC:       storage.MakeVolumeClaimTemplate(*pvcTemplate, req.Namespace),
	}
	mountEstablished, err := stor.ConvertVolume()
	if err != nil {
		return &ctrl.Result{}, fmt.Errorf("failed to mount on PVC: %v", err)
	}
	if mountEstablished { // this bool confirms that the deployment volume mount was updated. This bool does _not_ confirm that the deployment is mounted to the spec PVC.
		log.Info(fmt.Sprintf("deployment was successfully mounted onto PVC name: %s", stor.PVC.Name))
		return &ctrl.Result{}, nil
	}

	pvcStatus := &corev1.PersistentVolumeClaim{}
	namespace := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      pvcTemplate.Name}
	if err := r.Get(ctx, namespace, pvcStatus); err != nil {
		return &ctrl.Result{}, fmt.Errorf("failed to get PVC name %s, %v", pvcTemplate.Name, err)
	}
	cr.Status.PersistentVolumeClaim = storage.MakeEmbeddedPVC(pvcStatus)

	if strings.Contains(cr.Status.Storage.VolumeType, "EmptyDir") {
		cr.Status.Storage.VolumeMounted = false
		if err := r.Status().Update(ctx, cr); err != nil {
			log.Error(err, "failed to update MetricsConfig status")
		}
		return &ctrl.Result{}, fmt.Errorf("PVC not mounted")
	}
	return nil, nil
}

// concatErrs combines all the errors into one error
func concatErrs(errors ...error) error {
	var err error
	var errstrings []string
	for _, e := range errors {
		errstrings = append(errstrings, e.Error())
	}
	if len(errstrings) > 0 {
		err = fmt.Errorf(strings.Join(errstrings, "\n"))
	}
	return err
}

// +kubebuilder:rbac:groups=koku-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=kokumetricsconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=koku-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=kokumetricsconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,namespace=koku-metrics-operator,resources=clusterserviceversions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,namespace=koku-metrics-operator,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets;serviceaccounts,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=apps,namespace=koku-metrics-operator,resources=deployments,verbs=get;list;patch;watch

// Reconcile Process the MetricsConfig custom resource based on changes or requeue
func (r *MetricsConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	os.Setenv("TZ", "UTC")

	// fetch the MetricsConfig instance
	crOriginal := &metricscfgv1beta1.MetricsConfig{}

	if err := r.Get(ctx, req.NamespacedName, crOriginal); err != nil {
		log.Info(fmt.Sprintf("unable to fetch MetricsConfigCR: %v", err))
		// we'll ignore not-found errors, since they cannot be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	cr := crOriginal.DeepCopy()
	log.Info("reconciling custom resource", "MetricsConfig", cr)

	// reflect the spec values into status
	reflectSpec(r, cr)

	if r.InCluster {
		res, err := configurePVC(r, req, cr)
		if err != nil || res != nil {
			return *res, err
		}
	}

	// set the cluster ID & return if there are errors
	if err := setClusterID(r, cr); err != nil {
		log.Error(err, "failed to obtain clusterID")
		if err := r.Status().Update(ctx, cr); err != nil {
			log.Error(err, "failed to update MetricsConfig status")
		}
		return ctrl.Result{}, err
	}

	log.Info("using the following inputs", "MetricsConfigConfig", cr.Status)

	// set the Operator git commit and reflect it in the upload status
	setOperatorCommit(r)
	if cr.Status.OperatorCommit != GitCommit {
		// If the commit is different, this is either a fresh install or the operator was upgraded.
		// After an upgrade, the report structure may differ from the old report structure,
		// so we need to package the old files before generating new reports.
		// We set this packaging time to zero so that the next call to packageFiles
		// will force file packaging to occur.
		cr.Status.Packaging.LastSuccessfulPackagingTime = metav1.Time{}
		cr.Status.OperatorCommit = GitCommit
	}

	// Get or create the directory configuration
	log.Info("getting directory configuration")
	if dirCfg == nil || !dirCfg.CheckConfig() {
		if err := dirCfg.GetDirectoryConfig(); err != nil {
			log.Error(err, "failed to get directory configuration")
			return ctrl.Result{}, err // without this directory, it is pointless to continue
		}
	}

	packager := &packaging.FilePackager{
		CR:     cr,
		DirCfg: dirCfg,
	}

	// if packaging time is zero but there are files in the data dir, this is an upgraded operator.
	// package all the files so that the next prometheus query generates a fresh report
	if cr.Status.Packaging.LastSuccessfulPackagingTime.IsZero() && dirCfg != nil {
		log.Info("checking for files from an old operator version")
		files, err := dirCfg.Reports.GetFiles()
		if err == nil && len(files) > 0 {
			log.Info("packaging files from an old operator version")
			packageFiles(packager)
		}
	}

	// attempt to collect prometheus stats and create reports
	if err := getPromCollector(r, cr); err != nil {
		log.Error(err, "failed to get prometheus connection")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err // give things a break and try again in 2 minutes
	}
	originalStartTime, endTime := getTimeRange(ctx, r, cr)
	startTime := originalStartTime
	for startTime.Before(endTime) {
		t := startTime
		timeRange := promv1.Range{
			Start: t,
			End:   t.Add(59*time.Minute + 59*time.Second),
			Step:  time.Minute,
		}
		collectPromStats(r, cr, dirCfg, timeRange)
		if startTime.Sub(originalStartTime) == 48*time.Hour {
			// after collecting 48 hours of data, package the report to compress the files
			// packaging is guarded by this LastSuccessfulPackagingTime, so setting it to
			// zero enables packaging to occur thruout this loop
			cr.Status.Packaging.LastSuccessfulPackagingTime = metav1.Time{}
			packageFiles(packager)
			originalStartTime = startTime
		}
		startTime = startTime.Add(1 * time.Hour)
		if err := r.Status().Update(ctx, cr); err != nil {
			// it's not critical to handle this error. We update the status here to show progress
			// if this loop takes a long time to complete. A missed update here does not impact
			// data collection here.
			log.Info("failed to update MetricsConfig status")
		}
	}

	// package report files
	packageFiles(packager)

	// Initial returned result -> requeue reconcile after 5 min.
	// This result is replaced if upload or status update results in error.
	var result = ctrl.Result{RequeueAfter: time.Minute * 5}
	var errors []error

	if cr.Spec.Upload.UploadToggle != nil && *cr.Spec.Upload.UploadToggle {

		log.Info("configuration is for connected cluster")

		authConfig := &crhchttp.AuthConfig{
			ValidateCert:   *cr.Status.Upload.ValidateCert,
			Authentication: cr.Status.Authentication.AuthType,
			OperatorCommit: cr.Status.OperatorCommit,
			ClusterID:      cr.Status.ClusterID,
			Client:         r.Client,
		}

		// obtain credentials token/basic & return if there are authentication credential errors
		if err := setAuthentication(r, authConfig, cr, req.NamespacedName); err != nil {
			if err := r.Status().Update(ctx, cr); err != nil {
				log.Error(err, "failed to update MetricsConfig status")
			}
			return ctrl.Result{}, err
		}

		handler := &sources.SourceHandler{
			APIURL: cr.Status.APIURL,
			Auth:   authConfig,
			Spec:   cr.Status.Source,
		}

		if err := validateCredentials(r, handler, cr, 1440); err == nil {
			// Block will run when creds are valid.

			// Check if source is defined and update the status to confirmed/created
			checkSource(r, handler, cr)

			// attempt upload
			if err := uploadFiles(r, authConfig, cr, dirCfg, packager); err != nil {
				result = ctrl.Result{}
				errors = append(errors, err)
			}

			// revalidate if an upload fails due to 401
			if strings.Contains(cr.Status.Upload.LastUploadStatus, "401") {
				_ = validateCredentials(r, handler, cr, 0)
			}
		}
	} else {
		log.Info("configuration is for restricted-network cluster")
	}

	// remove old reports if maximum report count has been exceeded
	if err := packager.TrimPackages(); err != nil {
		result = ctrl.Result{}
		errors = append(errors, err)
	}

	uploadFiles, err := dirCfg.Upload.GetFilesFullPath()
	if err != nil {
		result = ctrl.Result{}
		errors = append(errors, err)
	}
	cr.Status.Packaging.PackagedFiles = uploadFiles

	if err := r.Status().Update(ctx, cr); err != nil {
		log.Error(err, "failed to update MetricsConfig status")
		result = ctrl.Result{}
		errors = append(errors, err)
	}

	// Requeue for processing after 5 minutes
	return result, concatErrs(errors...)
}

// SetupWithManager Setup reconciliation with manager object
func (r *MetricsConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metricscfgv1beta1.MetricsConfig{}).
		Complete(r)
}
