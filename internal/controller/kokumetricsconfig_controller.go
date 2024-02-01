//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controller

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
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	cv "github.com/project-koku/koku-metrics-operator/internal/clusterversion"
	"github.com/project-koku/koku-metrics-operator/internal/collector"
	"github.com/project-koku/koku-metrics-operator/internal/crhchttp"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/packaging"
	"github.com/project-koku/koku-metrics-operator/internal/sources"
	"github.com/project-koku/koku-metrics-operator/internal/storage"
)

const HOURS_IN_DAY int = 23 // first hour is 0: 0 -> 23 == 24 hrs

var (
	GitCommit string

	openShiftConfigNamespace = "openshift-config"
	pullSecretName           = "pull-secret"
	pullSecretDataKey        = ".dockerconfigjson"
	pullSecretAuthKey        = "cloud.openshift.com"
	authSecretUserKey        = "username"
	authSecretPasswordKey    = "password"
	authClientId             = "client_id"
	authClientSecret         = "client_secret"

	falseDef = false
	trueDef  = true

	dirCfg             *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	sourceSpec         *metricscfgv1beta1.CloudDotRedHatSourceSpec
	previousValidation *previousAuthValidation
	authConfig         *crhchttp.AuthConfig

	log = logr.Log.WithName("metricsconfig_controller")
)

// MetricsConfigReconciler reconciles a MetricsConfig object
type MetricsConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	InCluster bool
	Namespace string

	apiReader             client.Reader
	cvClientBuilder       cv.ClusterVersionBuilder
	promCollector         *collector.PrometheusCollector
	initialDataCollection bool
	overrideSecretPath    bool
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
func StringReflectSpec(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, specItem *string, statusItem *string, defaultVal string) (string, bool) {
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
func ReflectSpec(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) {

	if cr.Spec.APIURL == metricscfgv1beta1.OldDefaultAPIURL {
		defaultAPIURL := metricscfgv1beta1.DefaultAPIURL
		StringReflectSpec(r, cr, &defaultAPIURL, &cr.Status.APIURL, metricscfgv1beta1.DefaultAPIURL)
	} else {
		StringReflectSpec(r, cr, &cr.Spec.APIURL, &cr.Status.APIURL, metricscfgv1beta1.DefaultAPIURL)
	}
	StringReflectSpec(r, cr, &cr.Spec.Authentication.AuthenticationSecretName, &cr.Status.Authentication.AuthenticationSecretName, "")
	StringReflectSpec(r, cr, &cr.Spec.Authentication.TokenURL, &cr.Status.Authentication.TokenURL, metricscfgv1beta1.DefaultTokenURL)

	if !reflect.DeepEqual(cr.Spec.Authentication.AuthType, cr.Status.Authentication.AuthType) {
		cr.Status.Authentication.AuthType = cr.Spec.Authentication.AuthType
	}
	cr.Status.Upload.ValidateCert = cr.Spec.Upload.ValidateCert

	StringReflectSpec(r, cr, &cr.Spec.Upload.IngressAPIPath, &cr.Status.Upload.IngressAPIPath, metricscfgv1beta1.DefaultIngressPath)
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

	StringReflectSpec(r, cr, &cr.Spec.Source.SourcesAPIPath, &cr.Status.Source.SourcesAPIPath, metricscfgv1beta1.DefaultSourcesPath)
	StringReflectSpec(r, cr, &cr.Spec.Source.SourceName, &cr.Status.Source.SourceName, "")

	cr.Status.Source.CreateSource = cr.Spec.Source.CreateSource

	if !reflect.DeepEqual(cr.Spec.Source.CheckCycle, cr.Status.Source.CheckCycle) {
		cr.Status.Source.CheckCycle = cr.Spec.Source.CheckCycle
	}

	StringReflectSpec(r, cr, &cr.Spec.PrometheusConfig.SvcAddress, &cr.Status.Prometheus.SvcAddress, metricscfgv1beta1.DefaultPrometheusSvcAddress)
	cr.Status.Prometheus.SkipTLSVerification = cr.Spec.PrometheusConfig.SkipTLSVerification
	cr.Status.Prometheus.ContextTimeout = cr.Spec.PrometheusConfig.ContextTimeout
	cr.Status.Prometheus.DisabledMetricsCollectionCostManagement = cr.Spec.PrometheusConfig.DisableMetricsCollectionCostManagement
	cr.Status.Prometheus.DisabledMetricsCollectionResourceOptimization = cr.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization
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
func GetClusterID(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) error {
	log := log.WithName("GetClusterID")
	// Get current ClusterVersion
	cvClient := r.cvClientBuilder.New(r.Client)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}
	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		cr.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
	}
	if clusterVersion.Spec.Channel != "" {
		cr.Status.ClusterVersion = string(clusterVersion.Spec.Channel)
	}
	return nil
}

// LogSecretAccessError evaluates  the type of kube secret error and logs the appropriate message.
func LogSecretAccessError(err error, msg string) {
	switch {
	case errors.IsNotFound(err):
		errMsg := fmt.Sprintf("%s does not exist", msg)
		log.Error(err, errMsg)
	case errors.IsForbidden(err):
		errMsg := fmt.Sprintf("operator does not have permission to check %s", msg)
		log.Error(err, errMsg)
	default:
		errMsg := fmt.Sprintf("could not check %s", msg)
		log.Error(err, errMsg)
	}
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig) error {
	ctx := context.Background()
	log := log.WithName("GetPullSecretToken")

	secret, err := r.Clientset.CoreV1().Secrets(openShiftConfigNamespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	if err != nil {
		LogSecretAccessError(err, "pull-secret")
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
func GetAuthSecret(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := log.WithName("GetAuthSecret")

	if previousValidation == nil || previousValidation.secretName != cr.Status.Authentication.AuthenticationSecretName {
		previousValidation = &previousAuthValidation{secretName: cr.Status.Authentication.AuthenticationSecretName}
	}

	log.Info("secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      cr.Status.Authentication.AuthenticationSecretName}
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		LogSecretAccessError(err, "secret")
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

// GetServiceAccountSecret Obtain the client id and client secret from the service account data provided in the current namespace
func (r *MetricsConfigReconciler) GetServiceAccountSecret(ctx context.Context, cr *metricscfgv1beta1.MetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	log := log.WithName("GetServiceAccountSecret")

	// Fetching the Secret object
	secret := &corev1.Secret{}
	secretName := cr.Spec.Authentication.AuthenticationSecretName
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      secretName}

	log.Info("getting secret", "secret name", secretName, "namespace", reqNamespace.Namespace)
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		LogSecretAccessError(err, "service-account secret")
		return err
	}
	log.Info("found scecret")

	// Extracting data from the Secret
	keys := make(map[string]string)
	for k, v := range secret.Data {
		keys[strings.ToLower(k)] = string(v)
	}

	// Defining the required keys
	requiredKeys := []string{authClientId, authClientSecret}
	log.Info("getting keys from secret", "required keys", requiredKeys)
	for _, requiredKey := range requiredKeys {
		if len(keys[requiredKey]) <= 0 {
			msg := fmt.Sprintf("service account secret not found with expected %s data", requiredKey)
			log.Info(msg)
			return fmt.Errorf(msg)
		}
	}
	log.Info("found required keys in secret")

	// Populating the authConfig object
	authConfig.ServiceAccountData = crhchttp.ServiceAccountData{
		ClientID:     keys[authClientId],
		ClientSecret: keys[authClientSecret],
		GrantType:    "client_credentials",
	}

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

func setClusterID(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig) error {
	if cr.Status.ClusterID == "" || cr.Status.ClusterVersion == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err := GetClusterID(r, cr)
		return err
	}
	return nil
}

func (r *MetricsConfigReconciler) setAuthentication(ctx context.Context, authConfig *crhchttp.AuthConfig, cr *metricscfgv1beta1.MetricsConfig, reqNamespace types.NamespacedName) error {
	log := log.WithName("setAuthentication")
	cr.Status.Authentication.AuthenticationCredentialsFound = &trueDef
	if cr.Status.Authentication.AuthType == metricscfgv1beta1.Token {
		cr.Status.Authentication.ValidBasicAuth = nil
		cr.Status.Authentication.AuthErrorMessage = ""
		cr.Status.Authentication.LastVerificationTime = nil
		// Get token from pull secret
		err := GetPullSecretToken(r, authConfig)
		if err != nil {
			log.Error(nil, "failed to obtain cluster authentication token")
			cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cr.Status.Authentication.AuthErrorMessage = err.Error()
		}
		return err
	}

	if cr.Spec.Authentication.AuthenticationSecretName == "" {
		// No authentication secret name set when using basic or service-account auth
		cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
		err := fmt.Errorf("no authentication secret name set when using %s auth", cr.Status.Authentication.AuthType)
		cr.Status.Authentication.AuthErrorMessage = err.Error()
		cr.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}

	if cr.Status.Authentication.AuthType == metricscfgv1beta1.Basic {
		// Get user and password from auth secret in namespace
		err := GetAuthSecret(r, cr, authConfig, reqNamespace)
		if err != nil {
			log.Error(nil, "failed to obtain authentication secret credentials")
			cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cr.Status.Authentication.AuthErrorMessage = err.Error()
			cr.Status.Authentication.ValidBasicAuth = &falseDef
		}
		return err
	}

	cr.Status.Authentication.ValidBasicAuth = nil
	cr.Status.Authentication.LastVerificationTime = nil
	// Get client ID and client secret from service account secret
	err := r.GetServiceAccountSecret(ctx, cr, authConfig, reqNamespace)
	if err != nil {
		log.Error(nil, "failed to obtain service account secret credentials")
		cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
		cr.Status.Authentication.AuthErrorMessage = err.Error()
	}
	return err
}

func (r *MetricsConfigReconciler) validateCredentials(ctx context.Context, handler *sources.SourceHandler, cr *metricscfgv1beta1.MetricsConfig, cycle int64) error {
	log := log.WithName("validateCredentials")

	if cr.Spec.Authentication.AuthType == metricscfgv1beta1.Token {
		// no need to validate token auth
		return nil
	}

	// Service-account authentication check
	if cr.Spec.Authentication.AuthType == metricscfgv1beta1.ServiceAccount {
		if err := handler.Auth.GetAccessToken(ctx, cr.Spec.Authentication.TokenURL); err != nil {
			errorMsg := fmt.Sprintf("failed to obtain service-account token: %v", err)
			log.Info(errorMsg)
			cr.Status.Authentication.AuthErrorMessage = errorMsg
			return err
		}
		cr.Status.Authentication.AuthErrorMessage = ""
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

	cr.Status.Authentication.LastVerificationTime = &previousValidation.timestamp

	if err != nil && strings.Contains(err.Error(), "401") {
		msg := fmt.Sprintf("console.redhat.com credentials are invalid. Correct the username/password in `%s`. Updated credentials will be re-verified during the next reconciliation.", cr.Spec.Authentication.AuthenticationSecretName)
		log.Info(msg)
		cr.Status.Authentication.AuthErrorMessage = msg
		cr.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
	log.Info("credentials are valid")
	cr.Status.Authentication.AuthErrorMessage = ""
	cr.Status.Authentication.ValidBasicAuth = &trueDef
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

func packageFilesWithCycle(p *packaging.FilePackager, cr *metricscfgv1beta1.MetricsConfig) {
	log := log.WithName("packageAndUpload")

	// if its time to package
	if !checkCycle(log, *cr.Status.Upload.UploadCycle, cr.Status.Packaging.LastSuccessfulPackagingTime, "file packaging") {
		return
	}

	packageFiles(p, cr)
}

func packageFiles(p *packaging.FilePackager, cr *metricscfgv1beta1.MetricsConfig) {
	// Package and split the payload if necessary
	cr.Status.Packaging.PackagingError = ""
	if err := p.PackageReports(cr); err != nil {
		log.Error(err, "PackageReports failed")
		// update the CR packaging error status
		cr.Status.Packaging.PackagingError = err.Error()
	}
}

func filesToUpload(cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig) ([]string, error) {
	if !checkCycle(log, *cr.Status.Upload.UploadCycle, cr.Status.Upload.LastSuccessfulUploadTime, "upload") {
		return nil, nil
	}
	uploadFiles, err := dirCfg.Upload.GetFiles()
	if err != nil {
		log.Error(err, "failed to read upload directory")
		return nil, err
	}
	if len(uploadFiles) <= 0 {
		log.Info("no files to upload")
		return nil, nil
	}
	return uploadFiles, nil
}

func (r *MetricsConfigReconciler) uploadFiles(authConfig *crhchttp.AuthConfig, cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, packager *packaging.FilePackager, uploadFiles []string) error {
	log := log.WithName("uploadFiles")

	log.Info("files ready for upload: " + strings.Join(uploadFiles, ", "))
	log.Info(fmt.Sprintf("pausing for %d seconds before uploading", *cr.Status.Upload.UploadWait))
	time.Sleep(time.Duration(*cr.Status.Upload.UploadWait) * time.Second)
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
		ingressURL := cr.Status.APIURL + cr.Status.Upload.IngressAPIPath
		uploadStatus, uploadTime, requestID, err := crhchttp.Upload(authConfig, contentType, "POST", ingressURL, body, manifestInfo, file)
		cr.Status.Upload.LastUploadStatus = uploadStatus
		cr.Status.Upload.LastPayloadName = file
		cr.Status.Upload.LastPayloadFiles = manifestInfo.Files
		cr.Status.Upload.LastPayloadManifestID = manifestInfo.UUID
		cr.Status.Upload.LastPayloadRequestID = requestID
		cr.Status.Upload.UploadError = ""
		if err != nil {
			log.Error(err, "upload failed")
			cr.Status.Upload.UploadError = err.Error()
			return nil
		}
		if strings.Contains(uploadStatus, "202") {
			cr.Status.Upload.LastSuccessfulUploadTime = uploadTime
			// remove the tar.gz after a successful upload
			log.Info("removing tar file since upload was successful")
			if err := os.Remove(filepath.Join(dirCfg.Upload.Path, file)); err != nil {
				log.Error(err, "error removing tar file")
			}
		}
	}
	return nil
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
		r.updateStatusAndLogError(ctx, cr)
		return &ctrl.Result{}, fmt.Errorf("PVC not mounted")
	}
	return nil, nil
}

func (r *MetricsConfigReconciler) setAuthAndUpload(ctx context.Context, cr *metricscfgv1beta1.MetricsConfig, packager *packaging.FilePackager, req ctrl.Request) error {

	log.Info("configuration is for connected cluster")
	// `authConfig` carries state between reconciliations. The only time we should reset the AuthConfig
	// is when the AuthType changes between reconciliations, e.g. changing from basic auth to service-token
	if authConfig == nil || authConfig.Authentication != cr.Status.Authentication.AuthType {
		authConfig = &crhchttp.AuthConfig{
			Authentication: cr.Status.Authentication.AuthType,
			OperatorCommit: cr.Status.OperatorCommit,
			ClusterID:      cr.Status.ClusterID,
			Client:         r.Client,
		}
	}
	authConfig.ValidateCert = *cr.Status.Upload.ValidateCert

	// obtain credentials token/basic & return if there are authentication credential errors
	if err := r.setAuthentication(ctx, authConfig, cr, req.NamespacedName); err != nil {
		return err
	}

	uploadFiles, err := filesToUpload(cr, dirCfg)
	if err != nil {
		log.Error(err, "failed to get files to upload")
		return err
	}

	doUpload := uploadFiles != nil
	doSourceCheck := cr.Status.Source.LastSourceCheckTime.IsZero()

	if !doUpload && !doSourceCheck {
		// if there are no files and a time for source check, we do
		// not need to proceed. This will enable source creation
		// when the CR is first created.
		log.Info("no files to upload and skipping source check")
		return nil
	}

	handler := &sources.SourceHandler{
		APIURL: cr.Status.APIURL,
		Auth:   authConfig,
		Spec:   cr.Status.Source,
	}

	if err := r.validateCredentials(ctx, handler, cr, 1440); err != nil {
		log.Info("failed to validate credentials", "error", err)
		return err
	}
	// Check if source is defined and update the status to confirmed/created
	checkSource(r, handler, cr)

	if !doUpload {
		// only attempt upload when files are available to upload
		return nil
	}

	// attempt upload
	if err := r.uploadFiles(authConfig, cr, dirCfg, packager, uploadFiles); err != nil {
		log.Info("failed to upload files", "error", err)
		return err
	}

	// revalidate if an upload fails due to 401
	if strings.Contains(cr.Status.Upload.LastUploadStatus, "401") {
		return r.validateCredentials(ctx, handler, cr, 0)
	}

	return nil
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
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses/api,verbs=get;create;update

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
	ReflectSpec(r, cr)

	if r.InCluster {
		res, err := configurePVC(r, req, cr)
		if err != nil || res != nil {
			return *res, err
		}
	}

	// set the cluster ID & return if there are errors
	if err := setClusterID(r, cr); err != nil {
		log.Error(err, "failed to obtain clusterID")
		r.updateStatusAndLogError(ctx, cr)
		return ctrl.Result{}, err
	}

	// set cluster id as default source name
	if cr.Status.Source.SourceName == "" {
		log.Info("using cluster id as default source name")
		cr.Status.Source.SourceName = cr.Status.ClusterID
	}

	log.Info("using the following inputs", "MetricsConfigConfig", cr.Status)

	// set the Operator git commit and reflect it in the upload status
	newInstall := false
	setOperatorCommit(r)
	if cr.Status.OperatorCommit != GitCommit {
		// If the commit is different, this is either a fresh install or the operator was upgraded.
		// After an upgrade, the report structure may differ from the old report structure,
		// so we need to package the old files before generating new reports.
		// We set this packaging time to zero so that the next call to packageFiles
		// will force file packaging to occur.
		log.Info("git commit changed which indicates newly installed operator")
		cr.Status.OperatorCommit = GitCommit
		newInstall = true
	}

	// Get or create the directory configuration
	log.Info("getting directory configuration")
	if dirCfg == nil || !dirCfg.CheckConfig() {
		if err := dirCfg.GetDirectoryConfig(); err != nil {
			log.Error(err, "failed to get directory configuration")
			return ctrl.Result{}, err // without this directory, it is pointless to continue
		}
	}

	startTime, endTime := getTimeRange(ctx, r, cr)

	packager := &packaging.FilePackager{
		DirCfg:      dirCfg,
		FilesAction: packaging.MoveFiles,
	}

	// after upgrade, package all the files so that the next prometheus query generates a fresh report
	if newInstall && dirCfg != nil {
		log.Info("checking for files from an old operator version")
		files, err := dirCfg.Reports.GetFiles()
		if err == nil && len(files) > 0 {
			log.Info("packaging files from an old operator version")
			packageFiles(packager, cr)
			// after packaging files after an upgrade, truncate the start time so we recollect
			// all of today's data. This ensures that today's report contains any new report changes.
			startTime = startTime.Truncate(24 * time.Hour)
		}
	}

	// attempt to collect prometheus stats and create reports
	if err := getPromCollector(r, cr); err != nil {
		log.Info("failed to get prometheus connection", "error", err)
		r.updateStatusAndLogError(ctx, cr)
		return ctrl.Result{}, err
	}

	for start := startTime; !start.After(endTime); start = start.AddDate(0, 0, 1) {
		t := start
		hours := int(endTime.Sub(t).Hours())
		if hours > HOURS_IN_DAY {
			hours = HOURS_IN_DAY
		}
		for i := 0; i <= hours; i++ {
			timeRange := promv1.Range{
				Start: t,
				End:   t.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			if err := collectPromStats(r, cr, dirCfg, timeRange); err != nil {
				if err == collector.ErrNoData && t.Hour() == 0 && t.Day() != endTime.Day() && r.initialDataCollection {
					// if there is no data for the first hour of the day, and we are doing the
					// initial data collection, skip to the next day so we avoid collecting
					// partial data for a full day. This ensures we are generating a full daily
					// report upon initial ingest.
					log.Info("skipping data collection for day", "datetime", timeRange.Start)
					break
				}
			}
			t = t.Add(1 * time.Hour)
		}

		if r.initialDataCollection && t.Sub(startTime).Hours() == 96 {
			// only perform these steps during the initial data collection.
			// after collecting 96 hours of data, package the report to compress the files
			log.Info("collected 96 hours of data, packaging files")
			packageFiles(packager, cr)
			startTime = t
			// update status to show progress
			r.updateStatusAndLogError(ctx, cr)
		}
	}

	r.initialDataCollection = false
	packager.FilesAction = packaging.CopyFiles
	if endTime.Hour() == HOURS_IN_DAY {
		// when we've reached the end of the day. move the files so we stop appending to them
		packager.FilesAction = packaging.MoveFiles
		packageFiles(packager, cr)
	} else {
		// package report files
		packageFilesWithCycle(packager, cr)
	}

	// Initial returned result -> requeue reconcile after 5 min.
	// This result is replaced if upload or status update results in error.
	var result = ctrl.Result{RequeueAfter: time.Minute * 5}
	var errors []error

	if cr.Spec.Upload.UploadToggle != nil && *cr.Spec.Upload.UploadToggle {
		if err := r.setAuthAndUpload(ctx, cr, packager, req); err != nil {
			result = ctrl.Result{}
			errors = append(errors, err)
		}

	} else {
		log.Info("configuration is for restricted-network cluster")
	}

	// remove old reports if maximum report count has been exceeded
	if err := packager.TrimPackages(cr); err != nil {
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
		log.Info("failed to update MetricsConfig status", "error", err)
		result = ctrl.Result{}
		errors = append(errors, err)
	}

	// Requeue for processing after 5 minutes
	return result, concatErrs(errors...)
}

// SetupWithManager Setup reconciliation with manager object
func (r *MetricsConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.apiReader = mgr.GetAPIReader()
	return ctrl.NewControllerManagedBy(mgr).
		For(&metricscfgv1beta1.MetricsConfig{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(
				time.Duration(5*time.Second),
				time.Duration(5*time.Minute),
			)}).
		Complete(r)
}

func (r *MetricsConfigReconciler) updateStatusAndLogError(ctx context.Context, cr *metricscfgv1beta1.MetricsConfig) {
	if err := r.Status().Update(ctx, cr); err != nil {
		log.Info("failed to update MetricsConfig status", "error", err)
	}
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
