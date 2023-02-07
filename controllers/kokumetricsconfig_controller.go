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

	gologr "github.com/go-logr/logr"
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
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
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
	sourceSpec         *kokumetricscfgv1beta1.CloudDotRedHatSourceSpec
	previousValidation *previousAuthValidation
	promCfgSetter      collector.PrometheusConfigurationSetter = collector.SetPrometheusConfig
	promConnSetter     collector.PrometheusConnectionSetter    = collector.SetPrometheusConnection
	promConnTester     collector.PrometheusConnectionTester    = collector.TestPrometheusConnection

	log = logr.Log.WithName("controller_kokumetricsconfig")
)

// KokuMetricsConfigReconciler reconciles a KokuMetricsConfig object
type KokuMetricsConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	InCluster bool
	Namespace string

	cvClientBuilder               cv.ClusterVersionBuilder
	promCollector                 *collector.PrometheusCollector
	disablePreviousDataCollection bool
	overrideSecretPath            bool
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
func StringReflectSpec(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, specItem *string, statusItem *string, defaultVal string) (string, bool) {
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
func ReflectSpec(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) {

	StringReflectSpec(r, kmCfg, &kmCfg.Spec.APIURL, &kmCfg.Status.APIURL, kokumetricscfgv1beta1.DefaultAPIURL)
	StringReflectSpec(r, kmCfg, &kmCfg.Spec.Authentication.AuthenticationSecretName, &kmCfg.Status.Authentication.AuthenticationSecretName, "")

	if !reflect.DeepEqual(kmCfg.Spec.Authentication.AuthType, kmCfg.Status.Authentication.AuthType) {
		kmCfg.Status.Authentication.AuthType = kmCfg.Spec.Authentication.AuthType
	}
	kmCfg.Status.Upload.ValidateCert = kmCfg.Spec.Upload.ValidateCert

	StringReflectSpec(r, kmCfg, &kmCfg.Spec.Upload.IngressAPIPath, &kmCfg.Status.Upload.IngressAPIPath, kokumetricscfgv1beta1.DefaultIngressPath)
	kmCfg.Status.Upload.UploadToggle = kmCfg.Spec.Upload.UploadToggle

	// set the default max file size for packaging
	kmCfg.Status.Packaging.MaxSize = &kmCfg.Spec.Packaging.MaxSize
	kmCfg.Status.Packaging.MaxReports = &kmCfg.Spec.Packaging.MaxReports

	// set the upload wait to whatever is in the spec, if the spec is defined
	if kmCfg.Spec.Upload.UploadWait != nil {
		kmCfg.Status.Upload.UploadWait = kmCfg.Spec.Upload.UploadWait
	}

	// if the status is nil, generate an upload wait
	if kmCfg.Status.Upload.UploadWait == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		uploadWait := r.Int63() % 35
		kmCfg.Status.Upload.UploadWait = &uploadWait
	}

	if !reflect.DeepEqual(kmCfg.Spec.Upload.UploadCycle, kmCfg.Status.Upload.UploadCycle) {
		kmCfg.Status.Upload.UploadCycle = kmCfg.Spec.Upload.UploadCycle
	}

	StringReflectSpec(r, kmCfg, &kmCfg.Spec.Source.SourcesAPIPath, &kmCfg.Status.Source.SourcesAPIPath, kokumetricscfgv1beta1.DefaultSourcesPath)
	StringReflectSpec(r, kmCfg, &kmCfg.Spec.Source.SourceName, &kmCfg.Status.Source.SourceName, "")

	kmCfg.Status.Source.CreateSource = kmCfg.Spec.Source.CreateSource

	if !reflect.DeepEqual(kmCfg.Spec.Source.CheckCycle, kmCfg.Status.Source.CheckCycle) {
		kmCfg.Status.Source.CheckCycle = kmCfg.Spec.Source.CheckCycle
	}

	StringReflectSpec(r, kmCfg, &kmCfg.Spec.PrometheusConfig.SvcAddress, &kmCfg.Status.Prometheus.SvcAddress, kokumetricscfgv1beta1.DefaultPrometheusSvcAddress)
	kmCfg.Status.Prometheus.SkipTLSVerification = kmCfg.Spec.PrometheusConfig.SkipTLSVerification
	kmCfg.Status.Prometheus.ContextTimeout = kmCfg.Spec.PrometheusConfig.ContextTimeout
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
func GetClusterID(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) error {
	log := log.WithName("GetClusterID")
	// Get current ClusterVersion
	cvClient := r.cvClientBuilder.New(r.Client)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}
	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		kmCfg.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
	}
	if clusterVersion.Spec.Channel != "" {
		kmCfg.Status.ClusterVersion = string(clusterVersion.Spec.Channel)
	}
	return nil
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *KokuMetricsConfigReconciler, authConfig *crhchttp.AuthConfig) error {
	ctx := context.Background()
	log := log.WithName("GetPullSecretToken")

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
func GetAuthSecret(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := log.WithName("GetAuthSecret")

	if previousValidation == nil || previousValidation.secretName != kmCfg.Status.Authentication.AuthenticationSecretName {
		previousValidation = &previousAuthValidation{secretName: kmCfg.Status.Authentication.AuthenticationSecretName}
	}

	log.Info("secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      kmCfg.Status.Authentication.AuthenticationSecretName}
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

func setClusterID(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) error {
	if kmCfg.Status.ClusterID == "" || kmCfg.Status.ClusterVersion == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err := GetClusterID(r, kmCfg)
		return err
	}
	return nil
}

func setAuthentication(r *KokuMetricsConfigReconciler, authConfig *crhchttp.AuthConfig, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, reqNamespace types.NamespacedName) error {
	log := log.WithName("setAuthentication")
	kmCfg.Status.Authentication.AuthenticationCredentialsFound = &trueDef
	if kmCfg.Status.Authentication.AuthType == kokumetricscfgv1beta1.Token {
		kmCfg.Status.Authentication.ValidBasicAuth = nil
		kmCfg.Status.Authentication.AuthErrorMessage = ""
		kmCfg.Status.Authentication.LastVerificationTime = nil
		// Get token from pull secret
		err := GetPullSecretToken(r, authConfig)
		if err != nil {
			log.Error(nil, "failed to obtain cluster authentication token")
			kmCfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			kmCfg.Status.Authentication.AuthErrorMessage = err.Error()
		}
		return err
	} else if kmCfg.Spec.Authentication.AuthenticationSecretName != "" {
		// Get user and password from auth secret in namespace
		err := GetAuthSecret(r, kmCfg, authConfig, reqNamespace)
		if err != nil {
			log.Error(nil, "failed to obtain authentication secret credentials")
			kmCfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			kmCfg.Status.Authentication.AuthErrorMessage = err.Error()
			kmCfg.Status.Authentication.ValidBasicAuth = &falseDef
		}
		return err
	} else {
		// No authentication secret name set when using basic auth
		kmCfg.Status.Authentication.AuthenticationCredentialsFound = &falseDef
		err := fmt.Errorf("no authentication secret name set when using basic auth")
		kmCfg.Status.Authentication.AuthErrorMessage = err.Error()
		kmCfg.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
}

func validateCredentials(r *KokuMetricsConfigReconciler, handler *sources.SourceHandler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, cycle int64) error {
	log := log.WithName("validateCredentials")

	if kmCfg.Spec.Authentication.AuthType == kokumetricscfgv1beta1.Token {
		// no need to validate token auth
		return nil
	}

	if previousValidation == nil {
		previousValidation = &previousAuthValidation{}
	}

	if previousValidation.password == handler.Auth.BasicAuthPassword &&
		previousValidation.username == handler.Auth.BasicAuthUser &&
		!checkCycle(log, cycle, previousValidation.timestamp, "credential verification") {
		return previousValidation.err
	}

	log.Info("validating credentials")
	client := crhchttp.GetClient(handler.Auth)
	_, err := sources.GetSources(handler, client)

	previousValidation.username = handler.Auth.BasicAuthUser
	previousValidation.password = handler.Auth.BasicAuthPassword
	previousValidation.err = err
	previousValidation.timestamp = metav1.Now()

	kmCfg.Status.Authentication.LastVerificationTime = &previousValidation.timestamp

	if err != nil && strings.Contains(err.Error(), "401") {
		msg := fmt.Sprintf("cloud.redhat.com credentials are invalid. Correct the username/password in `%s`. Updated credentials will be re-verified during the next reconciliation.", kmCfg.Spec.Authentication.AuthenticationSecretName)
		log.Info(msg)
		kmCfg.Status.Authentication.AuthErrorMessage = msg
		kmCfg.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
	log.Info("credentials are valid")
	kmCfg.Status.Authentication.AuthErrorMessage = ""
	kmCfg.Status.Authentication.ValidBasicAuth = &trueDef
	return nil
}

func setOperatorCommit(r *KokuMetricsConfigReconciler) {
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

func checkSource(r *KokuMetricsConfigReconciler, handler *sources.SourceHandler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) {
	log := log.WithName("checkSource")

	// check if the Source Spec has changed
	updated := false
	if sourceSpec != nil {
		updated = !reflect.DeepEqual(*sourceSpec, kmCfg.Spec.Source)
	}
	sourceSpec = kmCfg.Spec.Source.DeepCopy()

	if handler.Spec.SourceName != "" && (updated || checkCycle(log, *handler.Spec.CheckCycle, handler.Spec.LastSourceCheckTime, "source check")) {
		client := crhchttp.GetClient(handler.Auth)
		kmCfg.Status.Source.SourceError = ""
		defined, lastCheck, err := sources.SourceGetOrCreate(handler, client)
		if err != nil {
			kmCfg.Status.Source.SourceError = err.Error()
			log.Info("source get or create message", "error", err)
		}
		kmCfg.Status.Source.SourceDefined = &defined
		kmCfg.Status.Source.LastSourceCheckTime = lastCheck
	}
}

func packageFiles(p *packaging.FilePackager) {
	log := log.WithName("packageAndUpload")

	// if its time to package
	if !checkCycle(log, *p.KMCfg.Status.Upload.UploadCycle, p.KMCfg.Status.Packaging.LastSuccessfulPackagingTime, "file packaging") {
		return
	}

	// Package and split the payload if necessary
	p.KMCfg.Status.Packaging.PackagingError = ""
	if err := p.PackageReports(); err != nil {
		log.Error(err, "PackageReports failed")
		// update the CR packaging error status
		p.KMCfg.Status.Packaging.PackagingError = err.Error()
	}
}

func uploadFiles(r *KokuMetricsConfigReconciler, authConfig *crhchttp.AuthConfig, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, dirCfg *dirconfig.DirectoryConfig, packager *packaging.FilePackager) error {
	log := log.WithName("uploadFiles")

	// if its time to upload/package
	if !*kmCfg.Spec.Upload.UploadToggle {
		log.Info("operator is configured to not upload reports")
		return nil
	}
	if !checkCycle(log, *kmCfg.Status.Upload.UploadCycle, kmCfg.Status.Upload.LastSuccessfulUploadTime, "upload") {
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
	log.Info("pausing for " + fmt.Sprintf("%d", *kmCfg.Status.Upload.UploadWait) + " seconds before uploading")
	time.Sleep(time.Duration(*kmCfg.Status.Upload.UploadWait) * time.Second)
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
		ingressURL := kmCfg.Status.APIURL + kmCfg.Status.Upload.IngressAPIPath
		uploadStatus, uploadTime, requestID, err := crhchttp.Upload(authConfig, contentType, "POST", ingressURL, body, manifestInfo, file)
		kmCfg.Status.Upload.LastUploadStatus = uploadStatus
		kmCfg.Status.Upload.LastPayloadName = file
		kmCfg.Status.Upload.LastPayloadFiles = manifestInfo.Files
		kmCfg.Status.Upload.LastPayloadManifestID = manifestInfo.UUID
		kmCfg.Status.Upload.LastPayloadRequestID = requestID
		kmCfg.Status.Upload.UploadError = ""
		if err != nil {
			log.Error(err, "upload failed")
			kmCfg.Status.Upload.UploadError = err.Error()
			return nil
		}
		if strings.Contains(uploadStatus, "202") {
			kmCfg.Status.Upload.LastSuccessfulUploadTime = uploadTime
			// remove the tar.gz after a successful upload
			log.Info("removing tar file since upload was successful")
			if err := os.Remove(filepath.Join(dirCfg.Upload.Path, file)); err != nil {
				log.Error(err, "error removing tar file")
			}
		}
	}
	return nil
}

func getTimeRange(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) (time.Time, time.Time) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour) // start of previous full hour
	end := start.Add(59*time.Minute + 59*time.Second)
	if kmCfg.Spec.PrometheusConfig.CollectPreviousData != nil &&
		*kmCfg.Spec.PrometheusConfig.CollectPreviousData &&
		kmCfg.Status.Prometheus.LastQuerySuccessTime.IsZero() &&
		!r.disablePreviousDataCollection {
		// LastQuerySuccessTime is zero when the CR is first created. We will only reset `start` to the first of the
		// month when the CR is first created, otherwise we stick to using the start of the previous full hour.
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
		kmCfg.Status.Prometheus.PreviousDataCollected = true
	}
	return start, end
}

func getPromCollector(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) error {
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
	r.promCollector.ContextTimeout = kmCfg.Spec.PrometheusConfig.ContextTimeout

	return r.promCollector.GetPromConn(kmCfg, promCfgSetter, promConnSetter, promConnTester)
}

func collectPromStats(r *KokuMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, dirCfg *dirconfig.DirectoryConfig, timeRange promv1.Range) {
	log := log.WithName("collectPromStats")

	r.promCollector.TimeSeries = &timeRange

	t := metav1.Time{Time: timeRange.Start}
	formattedStart := timeRange.Start.Format(time.RFC3339)
	formattedEnd := timeRange.End.Format(time.RFC3339)
	if kmCfg.Status.Prometheus.LastQuerySuccessTime.UTC().Format(promCompareFormat) == t.Format(promCompareFormat) {
		log.Info("reports already generated for range", "start", formattedStart, "end", formattedEnd)
		return
	}

	kmCfg.Status.Prometheus.LastQueryStartTime = t

	log.Info("generating reports for range", "start", formattedStart, "end", formattedEnd)
	if err := collector.GenerateReports(kmCfg, dirCfg, r.promCollector); err != nil {
		kmCfg.Status.Reports.DataCollected = false
		kmCfg.Status.Reports.DataCollectionMessage = fmt.Sprintf("error: %v", err)
		log.Error(err, "failed to generate reports")
		return
	}
	log.Info("reports generated for range", "start", formattedStart, "end", formattedEnd)
	kmCfg.Status.Prometheus.LastQuerySuccessTime = t
}

func configurePVC(r *KokuMetricsConfigReconciler, req ctrl.Request, kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig) (*ctrl.Result, error) {
	ctx := context.Background()
	log := log.WithName("configurePVC")
	pvcTemplate := kmCfg.Spec.VolumeClaimTemplate
	if pvcTemplate == nil {
		pvcTemplate = &storage.DefaultPVC
	}

	stor := &storage.Storage{
		Client:    r.Client,
		KMCfg:     kmCfg,
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
	kmCfg.Status.PersistentVolumeClaim = storage.MakeEmbeddedPVC(pvcStatus)

	if strings.Contains(kmCfg.Status.Storage.VolumeType, "EmptyDir") {
		kmCfg.Status.Storage.VolumeMounted = false
		if err := r.Status().Update(ctx, kmCfg); err != nil {
			log.Error(err, "failed to update KokuMetricsConfig status")
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

// Reconcile Process the KokuMetricsConfig custom resource based on changes or requeue
func (r *KokuMetricsConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	os.Setenv("TZ", "UTC")

	// fetch the KokuMetricsConfig instance
	kmCfgOriginal := &kokumetricscfgv1beta1.KokuMetricsConfig{}

	if err := r.Get(ctx, req.NamespacedName, kmCfgOriginal); err != nil {
		log.Info(fmt.Sprintf("unable to fetch KokuMetricsConfigCR: %v", err))
		// we'll ignore not-found errors, since they cannot be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	kmCfg := kmCfgOriginal.DeepCopy()
	log.Info("reconciling custom resource", "KokuMetricsConfig", kmCfg)

	// reflect the spec values into status
	ReflectSpec(r, kmCfg)

	if r.InCluster {
		res, err := configurePVC(r, req, kmCfg)
		if err != nil || res != nil {
			return *res, err
		}
	}

	// set the cluster ID & return if there are errors
	if err := setClusterID(r, kmCfg); err != nil {
		log.Error(err, "failed to obtain clusterID")
		if err := r.Status().Update(ctx, kmCfg); err != nil {
			log.Error(err, "failed to update KokuMetricsConfig status")
		}
		return ctrl.Result{}, err
	}

	log.Info("using the following inputs", "KokuMetricsConfigConfig", kmCfg.Status)

	// set the Operator git commit and reflect it in the upload status
	setOperatorCommit(r)
	if kmCfg.Status.OperatorCommit != GitCommit {
		// If the commit is different, this is either a fresh install or the operator was upgraded.
		// After an upgrade, the report structure may differ from the old report structure,
		// so we need to package the old files before generating new reports.
		// We set this packaging time to zero so that the next call to packageFiles
		// will force file packaging to occur.
		kmCfg.Status.Packaging.LastSuccessfulPackagingTime = metav1.Time{}
		kmCfg.Status.OperatorCommit = GitCommit
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
		KMCfg:  kmCfg,
		DirCfg: dirCfg,
	}

	// if packaging time is zero but there are files in the data dir, this is an upgraded operator.
	// package all the files so that the next prometheus query generates a fresh report
	if kmCfg.Status.Packaging.LastSuccessfulPackagingTime.IsZero() && dirCfg != nil {
		log.Info("checking for files from an old operator version")
		files, err := dirCfg.Reports.GetFiles()
		if err == nil && len(files) > 0 {
			log.Info("packaging files from an old operator version")
			packageFiles(packager)
		}
	}

	// attempt to collect prometheus stats and create reports
	if err := getPromCollector(r, kmCfg); err != nil {
		log.Error(err, "failed to get prometheus connection")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err // give things a break and try again in 2 minutes
	}
	originalStartTime, endTime := getTimeRange(r, kmCfg)
	startTime := originalStartTime
	for startTime.Before(endTime) {
		t := startTime
		timeRange := promv1.Range{
			Start: t,
			End:   t.Add(59*time.Minute + 59*time.Second),
			Step:  time.Minute,
		}
		collectPromStats(r, kmCfg, dirCfg, timeRange)
		if startTime.Sub(originalStartTime) == 48*time.Hour {
			// after collecting 48 hours of data, package the report to compress the files
			// packaging is guarded by this LastSuccessfulPackagingTime, so setting it to
			// zero enables packaging to occur thruout this loop
			kmCfg.Status.Packaging.LastSuccessfulPackagingTime = metav1.Time{}
			packageFiles(packager)
			originalStartTime = startTime
		}
		startTime = startTime.Add(1 * time.Hour)
	}

	// package report files
	packageFiles(packager)

	// Initial returned result -> requeue reconcile after 5 min.
	// This result is replaced if upload or status update results in error.
	var result = ctrl.Result{RequeueAfter: time.Minute * 5}
	var errors []error

	if kmCfg.Spec.Upload.UploadToggle != nil && *kmCfg.Spec.Upload.UploadToggle {

		log.Info("configuration is for connected cluster")

		authConfig := &crhchttp.AuthConfig{
			ValidateCert:   *kmCfg.Status.Upload.ValidateCert,
			Authentication: kmCfg.Status.Authentication.AuthType,
			OperatorCommit: kmCfg.Status.OperatorCommit,
			ClusterID:      kmCfg.Status.ClusterID,
			Client:         r.Client,
		}

		// obtain credentials token/basic & return if there are authentication credential errors
		if err := setAuthentication(r, authConfig, kmCfg, req.NamespacedName); err != nil {
			if err := r.Status().Update(ctx, kmCfg); err != nil {
				log.Error(err, "failed to update KokuMetricsConfig status")
			}
			return ctrl.Result{}, err
		}

		handler := &sources.SourceHandler{
			APIURL: kmCfg.Status.APIURL,
			Auth:   authConfig,
			Spec:   kmCfg.Status.Source,
		}

		if err := validateCredentials(r, handler, kmCfg, 1440); err == nil {
			// Block will run when creds are valid.

			// Check if source is defined and update the status to confirmed/created
			checkSource(r, handler, kmCfg)

			// attempt upload
			if err := uploadFiles(r, authConfig, kmCfg, dirCfg, packager); err != nil {
				result = ctrl.Result{}
				errors = append(errors, err)
			}

			// revalidate if an upload fails due to 401
			if strings.Contains(kmCfg.Status.Upload.LastUploadStatus, "401") {
				_ = validateCredentials(r, handler, kmCfg, 0)
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
	kmCfg.Status.Packaging.PackagedFiles = uploadFiles

	if err := r.Status().Update(ctx, kmCfg); err != nil {
		log.Error(err, "failed to update KokuMetricsConfig status")
		result = ctrl.Result{}
		errors = append(errors, err)
	}

	// Requeue for processing after 5 minutes
	return result, concatErrs(errors...)
}

// SetupWithManager Setup reconciliation with manager object
func (r *KokuMetricsConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kokumetricscfgv1beta1.KokuMetricsConfig{}).
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
