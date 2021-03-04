/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
)

// CostManagementMetricsConfigReconciler reconciles a CostManagementMetricsConfig object
type CostManagementMetricsConfigReconciler struct {
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
func StringReflectSpec(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, specItem *string, statusItem *string, defaultVal string) (string, bool) {
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
func ReflectSpec(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) {

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

// GetClusterID Collects the cluster identifier from the Cluster Version custom resource object
func GetClusterID(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) error {
	log := r.Log.WithValues("CostManagementMetricsConfig", "GetClusterID")
	// Get current ClusterVersion
	cvClient := r.cvClientBuilder.New(r)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}
	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		kmCfg.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
	}
	return nil
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *CostManagementMetricsConfigReconciler, authConfig *crhchttp.AuthConfig) error {
	ctx := context.Background()
	log := r.Log.WithValues("CostManagementMetricsConfig", "GetPullSecretToken")

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
func GetAuthSecret(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := r.Log.WithValues("CostManagementMetricsConfig", "GetAuthSecret")

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

func checkCycle(logger logr.Logger, cycle int64, lastExecution metav1.Time, action string) bool {
	log := logger.WithValues("CostManagementMetricsConfig", "checkCycle")
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

func setClusterID(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) error {
	if kmCfg.Status.ClusterID == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err := GetClusterID(r, kmCfg)
		return err
	}
	return nil
}

func setAuthentication(r *CostManagementMetricsConfigReconciler, authConfig *crhchttp.AuthConfig, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, reqNamespace types.NamespacedName) error {
	log := r.Log.WithValues("CostManagementMetricsConfig", "setAuthentication")
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

func validateCredentials(r *CostManagementMetricsConfigReconciler, sSpec *sources.SourceSpec, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, cycle int64) error {
	if kmCfg.Spec.Authentication.AuthType == kokumetricscfgv1beta1.Token {
		// no need to validate token auth
		return nil
	}

	log := r.Log.WithValues("CostManagementMetricsConfig", "validateCredentials")

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

func setOperatorCommit(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) {
	log := r.Log.WithName("setOperatorCommit")
	if GitCommit == "" {
		commit, exists := os.LookupEnv("GIT_COMMIT")
		if exists {
			msg := fmt.Sprintf("using git commit from environment: %s", commit)
			log.Info(msg)
			GitCommit = commit
		}
	}
	kmCfg.Status.OperatorCommit = GitCommit
}

func checkSource(r *CostManagementMetricsConfigReconciler, sSpec *sources.SourceSpec, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) {
	// check if the Source Spec has changed
	updated := false
	if sourceSpec != nil {
		updated = !reflect.DeepEqual(*sourceSpec, kmCfg.Spec.Source)
	}
	sourceSpec = kmCfg.Spec.Source.DeepCopy()

	log := r.Log.WithValues("CostManagementMetricsConfig", "checkSource")
	if sSpec.Spec.SourceName != "" && (updated || checkCycle(r.Log, *sSpec.Spec.CheckCycle, sSpec.Spec.LastSourceCheckTime, "source check")) {
		client := crhchttp.GetClient(sSpec.Auth)
		kmCfg.Status.Source.SourceError = ""
		defined, lastCheck, err := sources.SourceGetOrCreate(sSpec, client)
		if err != nil {
			kmCfg.Status.Source.SourceError = err.Error()
			log.Info("source get or create message", "error", err)
		}
		kmCfg.Status.Source.SourceDefined = &defined
		kmCfg.Status.Source.LastSourceCheckTime = lastCheck
	}
}

func packageFiles(p *packaging.FilePackager) {
	log := p.Log.WithValues("CostManagementMetricsConfig", "packageAndUpload")

	// if its time to package
	if !checkCycle(p.Log, *p.KMCfg.Status.Upload.UploadCycle, p.KMCfg.Status.Packaging.LastSuccessfulPackagingTime, "file packaging") {
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

func uploadFiles(r *CostManagementMetricsConfigReconciler, authConfig *crhchttp.AuthConfig, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, dirCfg *dirconfig.DirectoryConfig) error {
	log := r.Log.WithValues("costmanagementmetricsconfig", "uploadFiles")

	// if its time to upload/package
	if !*kmCfg.Spec.Upload.UploadToggle {
		log.Info("operator is configured to not upload reports")
		return nil
	}
	if !checkCycle(r.Log, *kmCfg.Status.Upload.UploadCycle, kmCfg.Status.Upload.LastSuccessfulUploadTime, "upload") {
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
		log.Info(fmt.Sprintf("uploading file: %s", file))
		// grab the body and the multipart file header
		body, contentType, err := crhchttp.GetMultiPartBodyAndHeaders(filepath.Join(dirCfg.Upload.Path, file))
		if err != nil {
			log.Error(err, "failed to set multipart body and headers")
			return err
		}
		ingressURL := kmCfg.Status.APIURL + kmCfg.Status.Upload.IngressAPIPath
		uploadStatus, uploadTime, err := crhchttp.Upload(authConfig, contentType, "POST", ingressURL, body)
		kmCfg.Status.Upload.LastUploadStatus = uploadStatus
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

func collectPromStats(r *CostManagementMetricsConfigReconciler, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, dirCfg *dirconfig.DirectoryConfig) {
	log := r.Log.WithValues("CostManagementMetricsConfig", "collectPromStats")
	if r.promCollector == nil {
		r.promCollector = &collector.PromCollector{
			Log:       r.Log,
			InCluster: r.InCluster,
		}
	}
	r.promCollector.TimeSeries = nil

	if err := r.promCollector.GetPromConn(kmCfg); err != nil {
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

	if kmCfg.Status.Prometheus.LastQuerySuccessTime.UTC().Format(promCompareFormat) == t.Format(promCompareFormat) {
		log.Info("reports already generated for range", "start", timeRange.Start, "end", timeRange.End)
		return
	}
	kmCfg.Status.Prometheus.LastQueryStartTime = t
	log.Info("generating reports for range", "start", timeRange.Start, "end", timeRange.End)
	if err := collector.GenerateReports(kmCfg, dirCfg, r.promCollector); err != nil {
		kmCfg.Status.Reports.DataCollected = false
		kmCfg.Status.Reports.DataCollectionMessage = fmt.Sprintf("error: %v", err)
		log.Error(err, "failed to generate reports")
		return
	}
	log.Info("reports generated for range", "start", timeRange.Start, "end", timeRange.End)
	kmCfg.Status.Prometheus.LastQuerySuccessTime = t
}

func configurePVC(r *CostManagementMetricsConfigReconciler, req ctrl.Request, kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) (*ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagementmetricsconfig", "configurePVC")
	pvcTemplate := kmCfg.Spec.VolumeClaimTemplate
	if pvcTemplate == nil {
		pvcTemplate = &storage.DefaultPVC
	}

	stor := &storage.Storage{
		Client:    r.Client,
		KMCfg:     kmCfg,
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
	kmCfg.Status.PersistentVolumeClaim = storage.MakeEmbeddedPVC(pvcStatus)

	if strings.Contains(kmCfg.Status.Storage.VolumeType, "EmptyDir") {
		kmCfg.Status.Storage.VolumeMounted = false
		if err := r.Status().Update(ctx, kmCfg); err != nil {
			log.Error(err, "failed to update CostManagementMetricsConfig status")
		}
		return &ctrl.Result{}, fmt.Errorf("PVC not mounted")
	}
	return nil, nil
}

// +kubebuilder:rbac:groups=cost-mgmt-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=costmanagementmetricsconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-mgmt-metrics-cfg.openshift.io,namespace=koku-metrics-operator,resources=costmanagementmetricsconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,namespace=koku-metrics-operator,resources=clusterserviceversions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get
// +kubebuilder:rbac:groups=core,namespace=koku-metrics-operator,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets;serviceaccounts,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=apps,namespace=koku-metrics-operator,resources=deployments,verbs=get;list;patch;watch

// Reconcile Process the CostManagementMetricsConfig custom resource based on changes or requeue
func (r *CostManagementMetricsConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	os.Setenv("TZ", "UTC")
	ctx := context.Background()
	log := r.Log.WithValues("CostManagementMetricsConfig", req.NamespacedName)

	// fetch the CostManagementMetricsConfig instance
	kmCfgOriginal := &kokumetricscfgv1beta1.CostManagementMetricsConfig{}

	if err := r.Get(ctx, req.NamespacedName, kmCfgOriginal); err != nil {
		log.Info(fmt.Sprintf("unable to fetch CostManagementMetricsConfigCR: %v", err))
		// we'll ignore not-found errors, since they cannot be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	kmCfg := kmCfgOriginal.DeepCopy()
	log.Info("reconciling custom resource", "CostManagementMetricsConfig", kmCfg)

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
			log.Error(err, "failed to update CostManagementMetricsConfig status")
		}
		return ctrl.Result{}, err
	}

	log.Info("using the following inputs", "CostManagementMetricsConfigConfig", kmCfg.Status)

	// set the Operator git commit and reflect it in the upload status & return if there are errors
	setOperatorCommit(r, kmCfg)

	// Get or create the directory configuration
	log.Info("getting directory configuration")
	if dirCfg == nil || !dirCfg.CheckConfig() {
		if err := dirCfg.GetDirectoryConfig(); err != nil {
			log.Error(err, "failed to get directory configuration")
			return ctrl.Result{}, err // without this directory, it is pointless to continue
		}
	}

	// attempt to collect prometheus stats and create reports
	collectPromStats(r, kmCfg, dirCfg)

	// package report files
	packager := &packaging.FilePackager{
		KMCfg:  kmCfg,
		DirCfg: dirCfg,
		Log:    r.Log,
	}
	packageFiles(packager)

	// Initial returned result -> requeue reconcile after 5 min.
	// This result is replaced if upload or status update results in error.
	var result = ctrl.Result{RequeueAfter: time.Minute * 5}
	var errors []error

	if kmCfg.Spec.Upload.UploadToggle != nil && *kmCfg.Spec.Upload.UploadToggle {

		log.Info("configuration is for connected cluster")

		authConfig := &crhchttp.AuthConfig{
			Log:            r.Log,
			ValidateCert:   *kmCfg.Status.Upload.ValidateCert,
			Authentication: kmCfg.Status.Authentication.AuthType,
			OperatorCommit: kmCfg.Status.OperatorCommit,
			ClusterID:      kmCfg.Status.ClusterID,
			Client:         r.Client,
		}

		// obtain credentials token/basic & return if there are authentication credential errors
		if err := setAuthentication(r, authConfig, kmCfg, req.NamespacedName); err != nil {
			if err := r.Status().Update(ctx, kmCfg); err != nil {
				log.Error(err, "failed to update CostManagementMetricsConfig status")
			}
			return ctrl.Result{}, err
		}

		sSpec := &sources.SourceSpec{
			APIURL: kmCfg.Status.APIURL,
			Auth:   authConfig,
			Spec:   kmCfg.Status.Source,
			Log:    r.Log,
		}

		if err := validateCredentials(r, sSpec, kmCfg, 1440); err == nil {
			// Block will run when creds are valid.

			// Check if source is defined and update the status to confirmed/created
			checkSource(r, sSpec, kmCfg)

			// attempt upload
			if err := uploadFiles(r, authConfig, kmCfg, dirCfg); err != nil {
				result = ctrl.Result{}
				errors = append(errors, err)
			}

			// revalidate if an upload fails due to 401
			if strings.Contains(kmCfg.Status.Upload.LastUploadStatus, "401") {
				_ = validateCredentials(r, sSpec, kmCfg, 0)
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
		log.Error(err, "failed to update CostManagementMetricsConfig status")
		result = ctrl.Result{}
		errors = append(errors, err)
	}

	// Requeue for processing after 5 minutes
	return result, concatErrs(errors...)
}

// SetupWithManager Setup reconciliation with manager object
func (r *CostManagementMetricsConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kokumetricscfgv1beta1.CostManagementMetricsConfig{}).
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
