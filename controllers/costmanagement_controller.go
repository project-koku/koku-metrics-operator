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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/xorcare/pointer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	cv "github.com/project-koku/korekuta-operator-go/clusterversion"
	"github.com/project-koku/korekuta-operator-go/collector"
	"github.com/project-koku/korekuta-operator-go/crhchttp"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	"github.com/project-koku/korekuta-operator-go/packaging"
	"github.com/project-koku/korekuta-operator-go/sources"
)

var (
	openShiftConfigNamespace = "openshift-config"
	pullSecretName           = "pull-secret"
	pullSecretDataKey        = ".dockerconfigjson"
	pullSecretAuthKey        = "cloud.openshift.com"
	authSecretUserKey        = "username"
	authSecretPasswordKey    = "password"

	dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
)

// CostManagementReconciler reconciles a CostManagement object
type CostManagementReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	cvClientBuilder cv.ClusterVersionBuilder
	promCollector   *collector.PromCollector
}

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}
type serializedAuth struct {
	Auth string `json:"auth"`
}

// StringReflectSpec Determine if the string Status item reflects the Spec item if not empty, otherwise take the default value.
func StringReflectSpec(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, specItem *string, statusItem *string, defaultVal string) (string, bool) {
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
func ReflectSpec(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, costConfig *crhchttp.CostManagementConfig) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "ReflectSpec")
	costConfig.APIURL, _ = StringReflectSpec(r, cost, &cost.Spec.APIURL, &cost.Status.APIURL, costmgmtv1alpha1.DefaultAPIURL)
	costConfig.AuthenticationSecretName, _ = StringReflectSpec(r, cost, &cost.Spec.Authentication.AuthenticationSecretName, &cost.Status.Authentication.AuthenticationSecretName, "")

	if cost.Status.Authentication.AuthType == "" || !reflect.DeepEqual(cost.Spec.Authentication.AuthType, cost.Status.Authentication.AuthType) {
		// If data is specified in the spec it should be used
		if cost.Spec.Authentication.AuthType != "" {
			cost.Status.Authentication.AuthType = cost.Spec.Authentication.AuthType
		} else {
			cost.Status.Authentication.AuthType = costmgmtv1alpha1.DefaultAuthenticationType
		}
	}
	costConfig.Authentication = cost.Status.Authentication.AuthType

	// If data is specified in the spec it should be used
	cost.Status.ValidateCert = cost.Spec.ValidateCert
	if cost.Status.ValidateCert != nil {
		costConfig.ValidateCert = *cost.Status.ValidateCert
	} else {
		costConfig.ValidateCert = costmgmtv1alpha1.DefaultValidateCert
	}

	costConfig.IngressAPIPath, _ = StringReflectSpec(r, cost, &cost.Spec.Upload.IngressAPIPath, &cost.Status.Upload.IngressAPIPath, costmgmtv1alpha1.DefaultIngressPath)
	cost.Status.Upload.UploadToggle = cost.Spec.Upload.UploadToggle
	if cost.Status.Upload.UploadToggle != nil {
		costConfig.UploadToggle = *cost.Status.Upload.UploadToggle
	} else {
		costConfig.UploadToggle = costmgmtv1alpha1.DefaultUploadToggle
	}

	// set the upload variables to what is in the struct
	costConfig.LastUploadStatus = cost.Status.Upload.LastUploadStatus
	costConfig.LastUploadTime = cost.Status.Upload.LastUploadTime
	costConfig.LastSuccessfulUploadTime = cost.Status.Upload.LastSuccessfulUploadTime

	// set the default max file size for packaging
	cost.Status.Packaging.MaxSize = cost.Spec.Packaging.MaxSize
	if cost.Status.Packaging.MaxSize != nil {
		costConfig.MaxSize = *cost.Status.Packaging.MaxSize
	} else {
		costConfig.MaxSize = costmgmtv1alpha1.DefaultMaxSize
	}

	if !reflect.DeepEqual(cost.Spec.Upload.UploadWait, cost.Status.Upload.UploadWait) {
		// If data is specified in the spec it should be used
		cost.Status.Upload.UploadWait = cost.Spec.Upload.UploadWait
	}
	if cost.Status.Upload.UploadWait != nil {
		costConfig.UploadWait = *cost.Status.Upload.UploadWait
	} else {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		costConfig.UploadWait = r.Int63() % 35
	}

	if !reflect.DeepEqual(cost.Spec.Upload.UploadCycle, cost.Status.Upload.UploadCycle) {
		cost.Status.Upload.UploadCycle = cost.Spec.Upload.UploadCycle
	}
	if cost.Status.Upload.UploadCycle != nil {
		costConfig.UploadCycle = *cost.Status.Upload.UploadCycle
	} else {
		costConfig.UploadCycle = costmgmtv1alpha1.DefaultUploadCycle
	}

	var sourceNameChanged bool
	costConfig.SourcesAPIPath, _ = StringReflectSpec(r, cost, &cost.Spec.Source.SourcesAPIPath, &cost.Status.Source.SourcesAPIPath, costmgmtv1alpha1.DefaultSourcesPath)
	costConfig.SourceName, sourceNameChanged = StringReflectSpec(r, cost, &cost.Spec.Source.SourceName, &cost.Status.Source.SourceName, "")

	createBefore := false
	if cost.Spec.Source.CreateSource != nil {
		costConfig.CreateSource = *cost.Spec.Source.CreateSource
		createBefore = *cost.Spec.Source.CreateSource
	}
	cost.Status.Source.CreateSource = &costConfig.CreateSource
	createChanged := !(createBefore == *cost.Status.Source.CreateSource)

	sourceCycleChange := false
	if !reflect.DeepEqual(cost.Spec.Source.CheckCycle, cost.Status.Source.CheckCycle) {
		cost.Status.Source.CheckCycle = cost.Spec.Source.CheckCycle
		sourceCycleChange = true
	}
	if cost.Status.Source.CheckCycle != nil {
		costConfig.SourceCheckCycle = *cost.Status.Source.CheckCycle
	} else {
		costConfig.SourceCheckCycle = costmgmtv1alpha1.DefaultSourceCheckCycle
	}
	if !sourceNameChanged && !sourceCycleChange && !createChanged {
		costConfig.LastSourceCheckTime = cost.Status.Source.LastSourceCheckTime
	}

	costConfig.PrometheusSvcAddress, _ = StringReflectSpec(r, cost, &cost.Spec.PrometheusConfig.SvcAddress, &cost.Status.Prometheus.SvcAddress, costmgmtv1alpha1.DefaultPrometheusSvcAddress)
	costConfig.LastQuerySuccessTime = cost.Status.Prometheus.LastQuerySuccessTime
	cost.Status.Prometheus.SkipTLSVerification = cost.Spec.PrometheusConfig.SkipTLSVerification
	if cost.Status.Prometheus.SkipTLSVerification == nil {
		cost.Status.Prometheus.SkipTLSVerification = pointer.Bool(false)
	}

	err := r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
		return err
	}
	return nil
}

// GetClusterID Collects the cluster identifier from the Cluster Version custom resource object
func GetClusterID(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, costConfig *crhchttp.CostManagementConfig) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "GetClusterID")
	// Get current ClusterVersion
	cvClient := r.cvClientBuilder.New(r)
	clusterVersion, err := cvClient.GetClusterVersion()
	if err != nil {
		return err
	}
	log.Info("cluster version found", "ClusterVersion", clusterVersion.Spec)
	if clusterVersion.Spec.ClusterID != "" {
		cost.Status.ClusterID = string(clusterVersion.Spec.ClusterID)
		costConfig.ClusterID = cost.Status.ClusterID
	}
	err = r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
		return err
	}
	return nil
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *CostManagementReconciler, costConfig *crhchttp.CostManagementConfig) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "GetPullSecretToken")
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: openShiftConfigNamespace,
		Name:      pullSecretName}
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "Pull-secret does not exist.")
		case errors.IsForbidden(err):
			log.Error(err, "Operator does not have permission to check pull-secret.")
		default:
			log.Error(err, "Could not check pull-secret.")
		}
		return err
	}

	tokenFound := false
	encodedPullSecret := secret.Data[pullSecretDataKey]
	if len(encodedPullSecret) <= 0 {
		return fmt.Errorf("Cluster authorization secret didn't have data.")
	}
	var pullSecret serializedAuthMap
	if err := json.Unmarshal(encodedPullSecret, &pullSecret); err != nil {
		log.Error(err, "Unable to unmarshal cluster pull-secret.")
		return err
	}
	if auth, ok := pullSecret.Auths[pullSecretAuthKey]; ok {
		token := strings.TrimSpace(auth.Auth)
		if strings.Contains(token, "\n") || strings.Contains(token, "\r") {
			return fmt.Errorf("Cluster authorization token is not valid: contains newlines.")
		}
		if len(token) > 0 {
			log.Info("Found cloud.openshift.com token.")
			costConfig.BearerTokenString = token
			tokenFound = true
		} else {
			return fmt.Errorf("Cluster authorization token is not found.")
		}
	} else {
		return fmt.Errorf("Cluster authorization token was not found in secret data.")
	}
	if !tokenFound {
		return fmt.Errorf("Cluster authorization token is not found.")
	}
	return nil
}

// GetAuthSecret Obtain the username and password from the authentication secret provided in the current namespace
func GetAuthSecret(r *CostManagementReconciler, costConfig *crhchttp.CostManagementConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "GetAuthSecret")

	log.Info("Secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      costConfig.AuthenticationSecretName}
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "Secret does not exist.")
		case errors.IsForbidden(err):
			log.Error(err, "Operator does not have permission to check secret.")
		default:
			log.Error(err, "Could not check secret.")
		}
		return err
	}

	if val, ok := secret.Data[authSecretUserKey]; ok {
		costConfig.BasicAuthUser = string(val)
	} else {
		log.Info("Secret not found with expected user data.")
		err = fmt.Errorf("Secret not found with expected user data.")
		return err
	}

	if val, ok := secret.Data[authSecretPasswordKey]; ok {
		costConfig.BasicAuthPassword = string(val)
	} else {
		log.Info("Secret not found with expected password data.")
		err = fmt.Errorf("Secret not found with expected password data.")
		return err
	}
	return nil
}

func checkCycle(logger logr.Logger, cycle int64, lastExecution metav1.Time, action string) bool {
	log := logger.WithValues("costmanagement", "checkCycle")
	if lastExecution.IsZero() {
		log.Info(fmt.Sprintf("There have been no prior successful %ss to cloud.redhat.com.", action))
		return true
	}

	duration := time.Since(lastExecution.Time.UTC())
	minutes := int64(duration.Minutes())
	log.Info(fmt.Sprintf("It has been %d minutes since the last successful %s.", minutes, action))
	if minutes >= cycle {
		log.Info(fmt.Sprintf("Executing %s to cloud.redhat.com.", action))
		return true
	}
	log.Info(fmt.Sprintf("Not time to execute the %s.", action))
	return false

}

// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies;networks,verbs=get;list
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews;tokenreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=list;watch
// +kubebuilder:rbac:groups=core,namespace=openshift-cost,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets,verbs=create;delete;get;list;patch;update;watch

// Reconcile Process the CostManagement custom resource based on changes or requeue
func (r *CostManagementReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	os.Setenv("TZ", "UTC")
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", req.NamespacedName)

	// Fetch the CostManagement instance
	cost := &costmgmtv1alpha1.CostManagement{}
	err := r.Get(ctx, req.NamespacedName, cost)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("CostManagement resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get CostManagement")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling custom resource", "CostManagement", cost)
	costConfig := &crhchttp.CostManagementConfig{Log: r.Log}
	err = ReflectSpec(r, cost, costConfig)
	if err != nil {
		log.Error(err, "Failed to update CostManagement status")
		return ctrl.Result{}, err
	}
	if costConfig.ClusterID == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err = GetClusterID(r, cost, costConfig)
		if err != nil {
			log.Error(err, "Failed to obtain clusterID.")
			return ctrl.Result{}, err
		}
	}
	log.Info("Using the following inputs", "CostManagementConfig", costConfig)

	// Obtain credentials token/basic
	if costConfig.Authentication == costmgmtv1alpha1.Token {
		// Get token from pull secret
		err = GetPullSecretToken(r, costConfig)
		if err != nil {
			log.Error(nil, "Failed to obtain cluster authentication token.")
			cost.Status.Authentication.AuthenticationCredentialsFound = pointer.Bool(false)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		} else {
			cost.Status.Authentication.AuthenticationCredentialsFound = pointer.Bool(true)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		}
	} else if costConfig.AuthenticationSecretName != "" {
		// Get user and password from auth secret in namespace
		err = GetAuthSecret(r, costConfig, req.NamespacedName)
		if err != nil {
			log.Error(nil, "Failed to obtain authentication secret credentials.")
			cost.Status.Authentication.AuthenticationCredentialsFound = pointer.Bool(false)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		} else {
			cost.Status.Authentication.AuthenticationCredentialsFound = pointer.Bool(true)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		}
	} else {
		// No authentication secret name set when using basic auth
		cost.Status.Authentication.AuthenticationCredentialsFound = pointer.Bool(false)
		err = r.Status().Update(ctx, cost)
		if err != nil {
			log.Error(err, "Failed to update CostManagement Status")
		}
		err = fmt.Errorf("No authentication secret name set when using basic auth.")
	}
	// returns if `Obtain credentials token/basic` errors
	if err != nil {
		return ctrl.Result{}, err
	}

	// Grab the Operator git commit and upload the status and input object with it
	commit, err := ioutil.ReadFile("commit")
	if err != nil {
		fmt.Println("File reading error", err)
		return ctrl.Result{}, err
	}
	cost.Status.OperatorCommit = strings.Replace(string(commit), "\n", "", -1)
	costConfig.OperatorCommit = cost.Status.OperatorCommit
	err = r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
	}

	// Check if source is defined and should be confirmed/created
	if costConfig.SourceName != "" && checkCycle(r.Log, costConfig.SourceCheckCycle, costConfig.LastSourceCheckTime, "source check") {
		cost.Status.Source.SourceError = ""
		defined, lastCheck, err := sources.SourceGetOrCreate(costConfig)
		if err != nil {
			cost.Status.Source.SourceError = err.Error()
			log.Info("source get or create message", "error", err)
		}
		cost.Status.Source.SourceDefined = &defined
		cost.Status.Source.LastSourceCheckTime = lastCheck
		if err := r.Status().Update(ctx, cost); err != nil {
			log.Error(err, "Failed to update CostManagement Status")
		}
	}

	log.Info("Getting directory configuration.")
	if dirCfg == nil || !dirCfg.Parent.Exists() {
		if err := dirCfg.GetDirectoryConfig(); err != nil {
			log.Error(err, "Failed to get directory configuration.")
		}
	}

	// if its time to upload/package
	if costConfig.UploadToggle && checkCycle(r.Log, costConfig.UploadCycle, costConfig.LastSuccessfulUploadTime, "upload") {
		// Package and split the payload if necessary
		packager := packaging.FilePackager{
			Cost:    cost,
			DirCfg:  dirCfg,
			Log:     r.Log,
			MaxSize: costConfig.MaxSize}
		if err := packager.PackageReports(); err != nil {
			log.Error(err, "PackageReports failed.")
			// update the CR packaging error status
			cost.Status.Packaging.PackagingError = err.Error()
			if err := r.Status().Update(ctx, cost); err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		} else {
			cost.Status.Packaging.PackagingError = ""
		}

		uploadFiles, err := packager.ReadUploadDir()
		if err != nil {
			log.Error(err, "Failed to read upload directory.")
		}

		if len(uploadFiles) > 0 {
			log.Info("Files ready for upload: " + strings.Join(uploadFiles, ", "))
			// Upload to c.rh.com
			var uploadStatus string
			var uploadTime metav1.Time

			log.Info("Pausing for " + fmt.Sprintf("%d", costConfig.UploadWait) + " seconds before uploading.")
			time.Sleep(time.Duration(costConfig.UploadWait) * time.Second)

			for _, file := range uploadFiles {
				if strings.Contains(file, "tar.gz") {
					log.Info(fmt.Sprintf("Uploading file: %s", file))
					// grab the body and the multipart file header
					body, contentType, err := crhchttp.GetMultiPartBodyAndHeaders(filepath.Join(dirCfg.Upload.Path, file))
					if err != nil {
						log.Error(err, "failed to set multipart body and headers")
						return ctrl.Result{}, err
					}
					ingressURL := costConfig.APIURL + costConfig.IngressAPIPath
					uploadStatus, uploadTime, err = crhchttp.Upload(costConfig, contentType, "POST", ingressURL, body)
					if err != nil {
						log.Error(err, "upload failed")
						return ctrl.Result{}, err
					}
					if uploadStatus != "" {
						cost.Status.Upload.LastUploadStatus = uploadStatus
						costConfig.LastUploadStatus = cost.Status.Upload.LastUploadStatus
						cost.Status.Upload.LastUploadTime = uploadTime
						costConfig.LastUploadTime = cost.Status.Upload.LastUploadTime
						if strings.Contains(uploadStatus, "202") {
							cost.Status.Upload.LastSuccessfulUploadTime = uploadTime
							costConfig.LastSuccessfulUploadTime = cost.Status.Upload.LastSuccessfulUploadTime
							// remove the tar.gz after a successful upload
							log.Info("Removing tar file since upload was successful!")
							if err := os.Remove(filepath.Join(dirCfg.Upload.Path, file)); err != nil {
								log.Error(err, "Error removing tar file")
							}
						}
						if err := r.Status().Update(ctx, cost); err != nil {
							log.Error(err, "Failed to update CostManagement Status")
						}
					}
				}
			}
		} else {
			log.Info("No files to upload.")
		}
	} else if !costConfig.UploadToggle {
		log.Info("Operator is configured to not upload reports to cloud.redhat.com!")
	}

	if r.promCollector == nil {
		r.promCollector = &collector.PromCollector{
			Client: r.Client,
			Log:    r.Log,
		}
	}
	r.promCollector.TimeSeries = nil

	if err := r.promCollector.GetPromConn(cost); err != nil {
		log.Error(err, "failed to get prometheus connection")
	} else {
		timeUTC := metav1.Now().UTC()
		t := metav1.Time{Time: timeUTC}
		timeRange := promv1.Range{
			Start: time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 0, 0, 0, t.Location()),
			End:   time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location()),
			Step:  time.Minute,
		}
		r.promCollector.TimeSeries = &timeRange
		if costConfig.LastQuerySuccessTime.IsZero() || costConfig.LastQuerySuccessTime.UTC().Hour() != t.Hour() {
			cost.Status.Prometheus.LastQueryStartTime = t
			log.Info("generating reports for range", "start", timeRange.Start, "end", timeRange.End)
			err = collector.GenerateReports(cost, dirCfg, r.promCollector)
			if err != nil {
				cost.Status.Reports.DataCollected = false
				cost.Status.Reports.DataCollectionMessage = fmt.Sprintf("Error: %v", err)
				log.Error(err, "failed to generate reports")
			} else {
				log.Info("reports generated for range", "start", timeRange.Start, "end", timeRange.End)
				cost.Status.Prometheus.LastQuerySuccessTime = t
			}
		} else {
			log.Info("reports already generated for range", "start", timeRange.Start, "end", timeRange.End)
		}
	}
	if err := r.Status().Update(ctx, cost); err != nil {
		log.Error(err, "failed to update CostManagement Status")
	}

	// Requeue for processing after 5 minutes
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// SetupWithManager Setup reconciliation with manager object
func (r *CostManagementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costmgmtv1alpha1.CostManagement{}).
		Complete(r)
}
