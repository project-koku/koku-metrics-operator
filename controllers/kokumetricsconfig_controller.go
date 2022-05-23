//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

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
	promCompareFormat        = "2006-01-02T15"

	falseDef = false
	trueDef  = true

	dirCfg             *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	sourceSpec         *metricscfgv1beta1.CloudDotRedHatSourceSpec
	previousValidation *previousAuthValidation
)

// MetricsConfigReconciler reconciles a MetricsConfig object
type MetricsConfigReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	InCluster bool
	Namespace string

	cvClientBuilder cv.ClusterVersionBuilder
	promCollector   *collector.PromCollector
}

type previousAuthValidation struct {
	secretName string
	username   string
	password   string
	err        error
	timestamp  metav1.Time
}

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}
type serializedAuth struct {
	Auth string `json:"auth"`
}

// StringReflectSpec Determine if the string Status item reflects the Spec item if not empty, otherwise take the default value.
func StringReflectSpec(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig, specItem *string, statusItem *string, defaultVal string) (string, bool) {
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

// ReflectSpec Determine if the Status item reflects the Spec item if not empty, otherwise set a default value if applicable.
func ReflectSpec(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig) {

	StringReflectSpec(r, cfg, &cfg.Spec.APIURL, &cfg.Status.APIURL, metricscfgv1beta1.DefaultAPIURL)
	StringReflectSpec(r, cfg, &cfg.Spec.Authentication.AuthenticationSecretName, &cfg.Status.Authentication.AuthenticationSecretName, "")

	if !reflect.DeepEqual(cfg.Spec.Authentication.AuthType, cfg.Status.Authentication.AuthType) {
		cfg.Status.Authentication.AuthType = cfg.Spec.Authentication.AuthType
	}
	cfg.Status.Upload.ValidateCert = cfg.Spec.Upload.ValidateCert

	StringReflectSpec(r, cfg, &cfg.Spec.Upload.IngressAPIPath, &cfg.Status.Upload.IngressAPIPath, metricscfgv1beta1.DefaultIngressPath)
	cfg.Status.Upload.UploadToggle = cfg.Spec.Upload.UploadToggle

	// set the default max file size for packaging
	cfg.Status.Packaging.MaxSize = &cfg.Spec.Packaging.MaxSize
	cfg.Status.Packaging.MaxReports = &cfg.Spec.Packaging.MaxReports

	// set the upload wait to whatever is in the spec, if the spec is defined
	if cfg.Spec.Upload.UploadWait != nil {
		cfg.Status.Upload.UploadWait = cfg.Spec.Upload.UploadWait
	}

	// if the status is nil, generate an upload wait
	if cfg.Status.Upload.UploadWait == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		uploadWait := r.Int63() % 35
		cfg.Status.Upload.UploadWait = &uploadWait
	}

	if !reflect.DeepEqual(cfg.Spec.Upload.UploadCycle, cfg.Status.Upload.UploadCycle) {
		cfg.Status.Upload.UploadCycle = cfg.Spec.Upload.UploadCycle
	}

	StringReflectSpec(r, cfg, &cfg.Spec.Source.SourcesAPIPath, &cfg.Status.Source.SourcesAPIPath, metricscfgv1beta1.DefaultSourcesPath)
	StringReflectSpec(r, cfg, &cfg.Spec.Source.SourceName, &cfg.Status.Source.SourceName, "")

	cfg.Status.Source.CreateSource = cfg.Spec.Source.CreateSource

	if !reflect.DeepEqual(cfg.Spec.Source.CheckCycle, cfg.Status.Source.CheckCycle) {
		cfg.Status.Source.CheckCycle = cfg.Spec.Source.CheckCycle
	}

	StringReflectSpec(r, cfg, &cfg.Spec.PrometheusConfig.SvcAddress, &cfg.Status.Prometheus.SvcAddress, metricscfgv1beta1.DefaultPrometheusSvcAddress)
	cfg.Status.Prometheus.SkipTLSVerification = cfg.Spec.PrometheusConfig.SkipTLSVerification
	cfg.Status.Prometheus.ContextTimeout = cfg.Spec.PrometheusConfig.ContextTimeout
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

// GetClusterID Collects the cluster identifier and version from the Cluster Version custom resource object
func GetClusterID(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig) error {
	log := r.Log.WithValues("MetricsConfig", "GetClusterID")
	// Get current ClusterVersion
	cvClient := r.cvClientBuilder.New(r.Client)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}
	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		cfg.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
	}
	if clusterVersion.Spec.Channel != "" {
		cfg.Status.ClusterVersion = string(clusterVersion.Spec.Channel)
	}
	return nil
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig) error {
	ctx := context.Background()
	log := r.Log.WithValues("MetricsConfig", "GetPullSecretToken")

	secret, err := r.Clientset.CoreV1().Secrets(openShiftConfigNamespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "pull-secret does not exist")
		case errors.IsForbidden(err):
			log.Error(err, "operator does not have permission to check pull-secret")
		default:
			log.Error(err, "could not check pull-secret")
		}
		return err
	}

	tokenFound := false
	encodedPullSecret := secret.Data[pullSecretDataKey]
	if len(encodedPullSecret) <= 0 {
		return fmt.Errorf("cluster authorization secret did not have data")
	}
	var pullSecret serializedAuthMap
	if err := json.Unmarshal(encodedPullSecret, &pullSecret); err != nil {
		log.Error(err, "unable to unmarshal cluster pull-secret")
		return err
	}
	if auth, ok := pullSecret.Auths[pullSecretAuthKey]; ok {
		token := strings.TrimSpace(auth.Auth)
		if strings.Contains(token, "\n") || strings.Contains(token, "\r") {
			return fmt.Errorf("cluster authorization token is not valid: contains newlines")
		}
		if len(token) > 0 {
			log.Info("found cloud.openshift.com token")
			authConfig.BearerTokenString = token
			tokenFound = true
		} else {
			return fmt.Errorf("cluster authorization token is not found")
		}
	} else {
		return fmt.Errorf("cluster authorization token was not found in secret data")
	}
	if !tokenFound {
		return fmt.Errorf("cluster authorization token is not found")
	}
	return nil
}

// GetAuthSecret Obtain the username and password from the authentication secret provided in the current namespace
func GetAuthSecret(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := r.Log.WithValues("MetricsConfig", "GetAuthSecret")

	if previousValidation == nil || previousValidation.secretName != cfg.Status.Authentication.AuthenticationSecretName {
		previousValidation = &previousAuthValidation{secretName: cfg.Status.Authentication.AuthenticationSecretName}
	}

	log.Info("secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      cfg.Status.Authentication.AuthenticationSecretName}
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "secret does not exist")
		case errors.IsForbidden(err):
			log.Error(err, "operator does not have permission to check secret")
		default:
			log.Error(err, "could not check secret")
		}
		return err
	}

	keys := make(map[string]string)
	for k, v := range secret.Data {
		keys[strings.ToLower(k)] = string(v)
	}

	for _, k := range []string{authSecretUserKey, authSecretPasswordKey} {
		if len(keys[k]) <= 0 {
			msg := fmt.Sprintf("secret not found with expected %s data", k)
			log.Info(msg)
			return fmt.Errorf(msg)
		}
	}

	authConfig.BasicAuthUser = keys[authSecretUserKey]
	authConfig.BasicAuthPassword = keys[authSecretPasswordKey]

	return nil
}

func checkCycle(logger logr.Logger, cycle int64, lastExecution metav1.Time, action string) bool {
	log := logger.WithValues("MetricsConfig", "checkCycle")
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

func setClusterID(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig) error {
	if cfg.Status.ClusterID == "" || cfg.Status.ClusterVersion == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err := GetClusterID(r, cfg)
		return err
	}
	return nil
}

func setAuthentication(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig, cfg *metricscfgv1beta1.MetricsConfig, reqNamespace types.NamespacedName) error {
	log := r.Log.WithValues("MetricsConfig", "setAuthentication")
	cfg.Status.Authentication.AuthenticationCredentialsFound = &trueDef
	if cfg.Status.Authentication.AuthType == metricscfgv1beta1.Token {
		cfg.Status.Authentication.ValidBasicAuth = nil
		cfg.Status.Authentication.AuthErrorMessage = ""
		cfg.Status.Authentication.LastVerificationTime = nil
		// Get token from pull secret
		err := GetPullSecretToken(r, authConfig)
		if err != nil {
			log.Error(nil, "failed to obtain cluster authentication token")
			cfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cfg.Status.Authentication.AuthErrorMessage = err.Error()
		}
		return err
	} else if cfg.Spec.Authentication.AuthenticationSecretName != "" {
		// Get user and password from auth secret in namespace
		err := GetAuthSecret(r, cfg, authConfig, reqNamespace)
		if err != nil {
			log.Error(nil, "failed to obtain authentication secret credentials")
			cfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cfg.Status.Authentication.AuthErrorMessage = err.Error()
			cfg.Status.Authentication.ValidBasicAuth = &falseDef
		}
		return err
	} else {
		// No authentication secret name set when using basic auth
		cfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
		err := fmt.Errorf("no authentication secret name set when using basic auth")
		cfg.Status.Authentication.AuthErrorMessage = err.Error()
		cfg.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
}

func validateCredentials(r *MetricsConfigReconciler, sSpec *sources.SourceSpec, cfg *metricscfgv1beta1.MetricsConfig, cycle int64) error {
	if cfg.Spec.Authentication.AuthType == metricscfgv1beta1.Token {
		// no need to validate token auth
		return nil
	}

	log := r.Log.WithValues("MetricsConfig", "validateCredentials")

	if previousValidation == nil {
		previousValidation = &previousAuthValidation{}
	}

	if previousValidation.password == sSpec.Auth.BasicAuthPassword &&
		previousValidation.username == sSpec.Auth.BasicAuthUser &&
		!checkCycle(r.Log, cycle, previousValidation.timestamp, "credential verification") {
		return previousValidation.err
	}

	log.Info("validating credentials")
	client := crhchttp.GetClient(sSpec.Auth)
	_, err := sources.GetSources(sSpec, client)

	previousValidation.username = sSpec.Auth.BasicAuthUser
	previousValidation.password = sSpec.Auth.BasicAuthPassword
	previousValidation.err = err
	previousValidation.timestamp = metav1.Now()

	cfg.Status.Authentication.LastVerificationTime = &previousValidation.timestamp

	if err != nil && strings.Contains(err.Error(), "401") {
		msg := fmt.Sprintf("cloud.redhat.com credentials are invalid. Correct the username/password in `%s`. Updated credentials will be re-verified during the next reconciliation.", cfg.Spec.Authentication.AuthenticationSecretName)
		log.Info(msg)
		cfg.Status.Authentication.AuthErrorMessage = msg
		cfg.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
	log.Info("credentials are valid")
	cfg.Status.Authentication.AuthErrorMessage = ""
	cfg.Status.Authentication.ValidBasicAuth = &trueDef
	return nil
}

func setOperatorCommit(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig) {
	log := r.Log.WithName("setOperatorCommit")
	if GitCommit == "" {
		commit, exists := os.LookupEnv("GIT_COMMIT")
		if exists {
			msg := fmt.Sprintf("using git commit from environment: %s", commit)
			log.Info(msg)
			GitCommit = commit
		}
	}
	if cfg.Status.OperatorCommit != GitCommit {
		// If the commit is different, this is either a fresh install or the operator was upgraded.
		// After an upgrade, the report structure may differ from the old report structure,
		// so we need to package the old files before generating new reports.
		// We set this packaging time to zero so that the next call to packageFiles
		// will force file packaging to occur.
		cfg.Status.Packaging.LastSuccessfulPackagingTime = metav1.Time{}
	}
	cfg.Status.OperatorCommit = GitCommit
}

func checkSource(r *MetricsConfigReconciler, sSpec *sources.SourceSpec, cfg *metricscfgv1beta1.MetricsConfig) {
	// check if the Source Spec has changed
	updated := false
	if sourceSpec != nil {
		updated = !reflect.DeepEqual(*sourceSpec, cfg.Spec.Source)
	}
	sourceSpec = cfg.Spec.Source.DeepCopy()

	log := r.Log.WithValues("MetricsConfig", "checkSource")
	if sSpec.Spec.SourceName != "" && (updated || checkCycle(r.Log, *sSpec.Spec.CheckCycle, sSpec.Spec.LastSourceCheckTime, "source check")) {
		client := crhchttp.GetClient(sSpec.Auth)
		cfg.Status.Source.SourceError = ""
		defined, lastCheck, err := sources.SourceGetOrCreate(sSpec, client)
		if err != nil {
			cfg.Status.Source.SourceError = err.Error()
			log.Info("source get or create message", "error", err)
		}
		cfg.Status.Source.SourceDefined = &defined
		cfg.Status.Source.LastSourceCheckTime = lastCheck
	}
}

func packageFiles(p *packaging.FilePackager) {
	log := p.Log.WithValues("MetricsConfig", "packageAndUpload")

	// if its time to package
	if !checkCycle(p.Log, *p.Cfg.Status.Upload.UploadCycle, p.Cfg.Status.Packaging.LastSuccessfulPackagingTime, "file packaging") {
		return
	}

	// Package and split the payload if necessary
	p.Cfg.Status.Packaging.PackagingError = ""
	if err := p.PackageReports(); err != nil {
		log.Error(err, "PackageReports failed")
		// update the CR packaging error status
		p.Cfg.Status.Packaging.PackagingError = err.Error()
	}
}

func uploadFiles(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig, cfg *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, packager *packaging.FilePackager) error {
	log := r.Log.WithValues("MetricsConfig", "uploadFiles")

	// if its time to upload/package
	if !*cfg.Spec.Upload.UploadToggle {
		log.Info("operator is configured to not upload reports")
		return nil
	}
	if !checkCycle(r.Log, *cfg.Status.Upload.UploadCycle, cfg.Status.Upload.LastSuccessfulUploadTime, "upload") {
		return nil
	}

	uploadFiles, err := dirCfg.Upload.GetFiles()
	if err != nil {
		log.Error(err, "failed to read upload directory")
		return err
	}

	if len(uploadFiles) <= 0 {
		log.Info("no files to upload")
		return nil
	}

	log.Info("files ready for upload: " + strings.Join(uploadFiles, ", "))
	log.Info("pausing for " + fmt.Sprintf("%d", *cfg.Status.Upload.UploadWait) + " seconds before uploading")
	time.Sleep(time.Duration(*cfg.Status.Upload.UploadWait) * time.Second)
	for _, file := range uploadFiles {
		if !strings.Contains(file, "tar.gz") {
			continue
		}

		manifestInfo, err := packager.GetFileInfo(filepath.Join(dirCfg.Upload.Path, file))
		if err != nil {
			log.Error(err, "Could not read file information from tar.gz")
			continue
		}

		log.Info(fmt.Sprintf("uploading file: %s", file))
		// grab the body and the multipart file header
		body, contentType, err := crhchttp.GetMultiPartBodyAndHeaders(filepath.Join(dirCfg.Upload.Path, file))
		if err != nil {
			log.Error(err, "failed to set multipart body and headers")
			return err
		}
		ingressURL := cfg.Status.APIURL + cfg.Status.Upload.IngressAPIPath
		uploadStatus, uploadTime, requestID, err := crhchttp.Upload(authConfig, contentType, "POST", ingressURL, body, manifestInfo, file)
		cfg.Status.Upload.LastUploadStatus = uploadStatus
		cfg.Status.Upload.LastPayloadName = file
		cfg.Status.Upload.LastPayloadFiles = manifestInfo.Files
		cfg.Status.Upload.LastPayloadManifestID = manifestInfo.UUID
		cfg.Status.Upload.LastPayloadRequestID = requestID
		cfg.Status.Upload.UploadError = ""
		if err != nil {
			log.Error(err, "upload failed")
			cfg.Status.Upload.UploadError = err.Error()
			return nil
		}
		if strings.Contains(uploadStatus, "202") {
			cfg.Status.Upload.LastSuccessfulUploadTime = uploadTime
			// remove the tar.gz after a successful upload
			log.Info("removing tar file since upload was successful")
			if err := os.Remove(filepath.Join(dirCfg.Upload.Path, file)); err != nil {
				log.Error(err, "error removing tar file")
			}
		}
	}
	return nil
}

func collectPromStats(r *MetricsConfigReconciler, cfg *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig) {
	log := r.Log.WithValues("MetricsConfig", "collectPromStats")
	if r.promCollector == nil {
		r.promCollector = &collector.PromCollector{
			Log:       r.Log,
			InCluster: r.InCluster,
		}
	}
	r.promCollector.TimeSeries = nil

	if err := r.promCollector.GetPromConn(cfg); err != nil {
		log.Error(err, "failed to get prometheus connection")
		return
	}
	timeUTC := metav1.Now().UTC()
	t := metav1.Time{Time: timeUTC}
	timeRange := promv1.Range{
		Start: time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 0, 0, 0, t.Location()),
		End:   time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location()),
		Step:  time.Minute,
	}
	r.promCollector.TimeSeries = &timeRange
	r.promCollector.ContextTimeout = cfg.Spec.PrometheusConfig.ContextTimeout

	if cfg.Status.Prometheus.LastQuerySuccessTime.UTC().Format(promCompareFormat) == t.Format(promCompareFormat) {
		log.Info("reports already generated for range", "start", timeRange.Start, "end", timeRange.End)
		return
	}
	cfg.Status.Prometheus.LastQueryStartTime = t
	log.Info("generating reports for range", "start", timeRange.Start, "end", timeRange.End)
	if err := collector.GenerateReports(cfg, dirCfg, r.promCollector); err != nil {
		cfg.Status.Reports.DataCollected = false
		cfg.Status.Reports.DataCollectionMessage = fmt.Sprintf("error: %v", err)
		log.Error(err, "failed to generate reports")
		return
	}
	log.Info("reports generated for range", "start", timeRange.Start, "end", timeRange.End)
	cfg.Status.Prometheus.LastQuerySuccessTime = t
}

func configurePVC(r *MetricsConfigReconciler, req ctrl.Request, cfg *metricscfgv1beta1.MetricsConfig) (*ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("MetricsConfig", "configurePVC")
	pvcTemplate := cfg.Spec.VolumeClaimTemplate
	if pvcTemplate == nil {
		pvcTemplate = &storage.DefaultPVC
	}

	stor := &storage.Storage{
		Client:    r.Client,
		Cfg:       cfg,
		Log:       r.Log,
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
	cfg.Status.PersistentVolumeClaim = storage.MakeEmbeddedPVC(pvcStatus)

	if strings.Contains(cfg.Status.Storage.VolumeType, "EmptyDir") {
		cfg.Status.Storage.VolumeMounted = false
		if err := r.Status().Update(ctx, cfg); err != nil {
			log.Error(err, "failed to update MetricsConfig status")
		}
		return &ctrl.Result{}, fmt.Errorf("PVC not mounted")
	}
	return nil, nil
}

// +kubebuilder:rbac:groups=koku-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=kokumetricsconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=koku-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=kokumetricsconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,namespace=koku-metrics-operator,resources=clusterserviceversions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get
// +kubebuilder:rbac:groups=core,namespace=koku-metrics-operator,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets;serviceaccounts,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=apps,namespace=koku-metrics-operator,resources=deployments,verbs=get;list;patch;watch

// Reconcile Process the MetricsConfig custom resource based on changes or requeue
func (r *MetricsConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	os.Setenv("TZ", "UTC")
	log := r.Log.WithValues("MetricsConfig", req.NamespacedName)

	// fetch the MetricsConfig instance
	cfgOriginal := &metricscfgv1beta1.KokuMetricsConfig{}

	if err := r.Get(ctx, req.NamespacedName, cfgOriginal); err != nil {
		log.Info(fmt.Sprintf("unable to fetch KokuMetricsConfigCR: %v", err))
		// we'll ignore not-found errors, since they cannot be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	cfg := &metricscfgv1beta1.MetricsConfig{cfgOriginal.DeepCopy()}
	log.Info("reconciling custom resource", "MetricsConfig", cfg)

	// reflect the spec values into status
	ReflectSpec(r, cfg)

	if r.InCluster {
		res, err := configurePVC(r, req, cfg)
		if err != nil || res != nil {
			return *res, err
		}
	}

	// set the cluster ID & return if there are errors
	if err := setClusterID(r, cfg); err != nil {
		log.Error(err, "failed to obtain clusterID")
		if err := r.Status().Update(ctx, cfg); err != nil {
			log.Error(err, "failed to update MetricsConfig status")
		}
		return ctrl.Result{}, err
	}

	log.Info("using the following inputs", "MetricsConfig", cfg.Status)

	// set the Operator git commit and reflect it in the upload status & return if there are errors
	setOperatorCommit(r, cfg)

	// Get or create the directory configuration
	log.Info("getting directory configuration")
	if dirCfg == nil || !dirCfg.CheckConfig() {
		if err := dirCfg.GetDirectoryConfig(); err != nil {
			log.Error(err, "failed to get directory configuration")
			return ctrl.Result{}, err // without this directory, it is pointless to continue
		}
	}

	packager := &packaging.FilePackager{
		Cfg:    cfg,
		DirCfg: dirCfg,
		Log:    r.Log,
	}

	// if packaging time is zero but there are files in the data dir, this is an upgraded operator.
	// package all the files so that the next prometheus query generates a fresh report
	if cfg.Status.Packaging.LastSuccessfulPackagingTime.IsZero() && dirCfg != nil {
		log.Info("checking for files from an old operator version")
		files, err := dirCfg.Reports.GetFiles()
		if err == nil && len(files) > 0 {
			log.Info("packaging files from an old operator version")
			packageFiles(packager)
		}
	}

	// attempt to collect prometheus stats and create reports
	collectPromStats(r, cfg, dirCfg)

	// package report files
	packageFiles(packager)

	// Initial returned result -> requeue reconcile after 5 min.
	// This result is replaced if upload or status update results in error.
	var result = ctrl.Result{RequeueAfter: time.Minute * 5}
	var errors []error

	if cfg.Spec.Upload.UploadToggle != nil && *cfg.Spec.Upload.UploadToggle {

		log.Info("configuration is for connected cluster")

		authConfig := &crhchttp.AuthConfig{
			Log:            r.Log,
			ValidateCert:   *cfg.Status.Upload.ValidateCert,
			Authentication: cfg.Status.Authentication.AuthType,
			OperatorCommit: cfg.Status.OperatorCommit,
			ClusterID:      cfg.Status.ClusterID,
			Client:         r.Client,
		}

		// obtain credentials token/basic & return if there are authentication credential errors
		if err := setAuthentication(r, authConfig, cfg, req.NamespacedName); err != nil {
			if err := r.Status().Update(ctx, cfg); err != nil {
				log.Error(err, "failed to update MetricsConfig status")
			}
			return ctrl.Result{}, err
		}

		sSpec := &sources.SourceSpec{
			APIURL: cfg.Status.APIURL,
			Auth:   authConfig,
			Spec:   cfg.Status.Source,
			Log:    r.Log,
		}

		if err := validateCredentials(r, sSpec, cfg, 1440); err == nil {
			// Block will run when creds are valid.

			// Check if source is defined and update the status to confirmed/created
			checkSource(r, sSpec, cfg)

			// attempt upload
			if err := uploadFiles(r, authConfig, cfg, dirCfg, packager); err != nil {
				result = ctrl.Result{}
				errors = append(errors, err)
			}

			// revalidate if an upload fails due to 401
			if strings.Contains(cfg.Status.Upload.LastUploadStatus, "401") {
				_ = validateCredentials(r, sSpec, cfg, 0)
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
	cfg.Status.Packaging.PackagedFiles = uploadFiles

	if err := r.Status().Update(ctx, cfg); err != nil {
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
		For(&metricscfgv1beta1.KokuMetricsConfig{}).
		Complete(r)
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
