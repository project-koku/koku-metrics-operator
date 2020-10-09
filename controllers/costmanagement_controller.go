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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
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
)

var (
	openShiftConfigNamespace = "openshift-config"
	pullSecretName           = "pull-secret"
	pullSecretDataKey        = ".dockerconfigjson"
	pullSecretAuthKey        = "cloud.openshift.com"
	authSecretUserKey        = "username"
	authSecretPasswordKey    = "password"
)

// CostManagementReconciler reconciles a CostManagement object
type CostManagementReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	cvClientBuilder cv.ClusterVersionBuilder
}

// CostManagementInput provide the data for procesing the reconcile with defaults
type CostManagementInput struct {
	ClusterID                string
	ValidateCert             bool
	IngressURL               string
	AuthenticationSecretName string
	Authentication           costmgmtv1alpha1.AuthenticationType
	UploadWait               int64
	BearerTokenString        string
	BasicAuthUser            string
	BasicAuthPassword        string
	LastUploadStatus         string
	LastUploadTime           string
	LastSuccessfulUploadTime string
	PrometheusConnected      bool
	LastQueryStartTime       metav1.Time
	LastQuerySuccessTime     metav1.Time
	OperatorCommit           string
	SourceName               string
	CreateSource             bool
}

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}
type serializedAuth struct {
	Auth string `json:"auth"`
}

// StringReflectSpec Determine if the string Status item reflects the Spec item if not empty, otherwise take the default value.
func StringReflectSpec(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, specItem *string, statusItem *string, defaultVal string) string {
	// Update statusItem if needed
	if *statusItem == "" || !reflect.DeepEqual(*specItem, *statusItem) {
		// If data is specified in the spec it should be used
		if *specItem != "" {
			*statusItem = *specItem
		} else if defaultVal != "" {
			*statusItem = defaultVal
		} else {
			*statusItem = *specItem
		}
	}
	return *statusItem
}

// ReflectSpec Determine if the Status item reflects the Spec item if not empty, otherwise set a default value if applicable.
func ReflectSpec(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, costInput *CostManagementInput) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "ReflectSpec")
	costInput.IngressURL = StringReflectSpec(r, cost, &cost.Spec.IngressURL, &cost.Status.IngressURL, costmgmtv1alpha1.DefaultIngressURL)
	costInput.AuthenticationSecretName = StringReflectSpec(r, cost, &cost.Spec.Authentication.AuthenticationSecretName, &cost.Status.Authentication.AuthenticationSecretName, "")

	if cost.Status.Authentication.AuthType == "" || !reflect.DeepEqual(cost.Spec.Authentication.AuthType, cost.Status.Authentication.AuthType) {
		// If data is specified in the spec it should be used
		if cost.Spec.Authentication.AuthType != "" {
			cost.Status.Authentication.AuthType = cost.Spec.Authentication.AuthType
		} else {
			cost.Status.Authentication.AuthType = costmgmtv1alpha1.DefaultAuthenticationType
		}
	}
	costInput.Authentication = cost.Status.Authentication.AuthType

	// If data is specified in the spec it should be used
	cost.Status.ValidateCert = cost.Spec.ValidateCert
	if cost.Status.ValidateCert != nil {
		costInput.ValidateCert = *cost.Status.ValidateCert
	} else {
		costInput.ValidateCert = costmgmtv1alpha1.DefaultValidateCert
	}

	if !reflect.DeepEqual(cost.Spec.UploadWait, cost.Status.UploadWait) {
		// If data is specified in the spec it should be used
		cost.Status.UploadWait = cost.Spec.UploadWait
	}
	if cost.Status.UploadWait != nil {
		costInput.UploadWait = *cost.Status.UploadWait
	} else {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		costInput.UploadWait = r.Int63() % 35
	}

	costInput.SourceName = StringReflectSpec(r, cost, &cost.Spec.Source.SourceName, &cost.Status.Source.SourceName, "")
	costInput.CreateSource = false
	if cost.Spec.Source.CreateSource != nil {
		costInput.CreateSource = *cost.Spec.Source.CreateSource
	}

	err := r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
		return err
	}
	return nil
}

// GetClusterID Collects the cluster identifier from the Cluster Version custom resource object
func GetClusterID(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, costInput *CostManagementInput) error {
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
		costInput.ClusterID = cost.Status.ClusterID
	}
	err = r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
		return err
	}
	return nil
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func GetPullSecretToken(r *CostManagementReconciler, costInput *CostManagementInput) error {
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
			costInput.BearerTokenString = token
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
func GetAuthSecret(r *CostManagementReconciler, costInput *CostManagementInput, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "GetAuthSecret")

	log.Info("Secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      costInput.AuthenticationSecretName}
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
		costInput.BasicAuthUser = string(val)
	} else {
		log.Info("Secret not found with expected user data.")
		err = fmt.Errorf("Secret not found with expected user data.")
		return err
	}

	if val, ok := secret.Data[authSecretPasswordKey]; ok {
		costInput.BasicAuthPassword = string(val)
	} else {
		log.Info("Secret not found with expected password data.")
		err = fmt.Errorf("Secret not found with expected password data.")
		return err
	}
	return nil
}

func GetBodyAndHeaders(r *CostManagementReconciler, filename string) (*bytes.Buffer, *multipart.Writer) {
	log := r.Log.WithValues("costmanagement", "GetBodyAndHeaders")
	// set the content and content type
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", filename))
	h.Set("Content-Type", "application/vnd.redhat.hccm.tar+tgz")
	fw, err := mw.CreatePart(h)
	f, err := os.Open(filename)
	if err != nil {
		log.Info("error opening file", err)
	}
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		log.Error(err, "The following error occurred")
	}
	mw.Close()
	return buf, mw
}

func Upload(r *CostManagementReconciler, costInput *CostManagementInput, method string, path string, body *bytes.Buffer, mw *multipart.Writer) (string, string, error) {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "Upload")
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		log.Error(err, "Could not create request")
		return "", "", err
	}
	// Create the header
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if costInput.Authentication == "basic" {
		log.Info("Uploading using basic authentication!")
		req.SetBasicAuth(costInput.BasicAuthUser, costInput.BasicAuthPassword)
	} else {
		log.Info("Uploading using token authentication")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", costInput.BearerTokenString))
		req.Header.Set("User-Agent", fmt.Sprintf("cost-mgmt-operator/%s cluster/%s", costInput.OperatorCommit, costInput.ClusterID))
	}
	// Log the headers - probably remove this later
	log.Info("Request Headers:")
	for key, val := range req.Header {
		fmt.Println(key, val)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return "", "", err
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))
	uploadStatus := fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	uploadTime := time.Now()

	// Add error handling and logging here
	requestID := resp.Header.Get("x-rh-insights-request-id")
	if resp.StatusCode == http.StatusUnauthorized {
		log.Info(fmt.Sprintf("gateway server %s returned 401, x-rh-insights-request-id=%s", resp.Request.URL, requestID))
	}
	if resp.StatusCode == http.StatusForbidden {
		log.Info(fmt.Sprintf("gateway server %s returned 403, x-rh-insights-request-id=%s", resp.Request.URL, requestID))
	}
	if resp.StatusCode == http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) > 1024 {
			body = body[:1024]
		}
		log.Info(fmt.Sprintf("gateway server bad request: %s (request=%s): %s", resp.Request.URL, requestID, string(body)))
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) > 1024 {
			body = body[:1024]
		}
		log.Info(fmt.Sprintf("gateway server reported unexpected error code: %d (request=%s): %s", resp.StatusCode, requestID, string(body)))
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Info(fmt.Sprintf("Successfully uploaded x-rh-insights-request-id=%s", requestID))
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "The following error occurred")
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	return uploadStatus, uploadTime.Format("2006-01-02 15:04:05"), err
}

// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies;networks,verbs=get;list
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews;tokenreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=list;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,namespace=openshift-cost,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets,verbs=create;delete;get;list;patch;update;watch

// Reconcile Process the CostManagement custom resource based on changes or requeue
func (r *CostManagementReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
	costInput := &CostManagementInput{}
	err = ReflectSpec(r, cost, costInput)
	if err != nil {
		log.Error(err, "Failed to update CostManagement status")
		return ctrl.Result{}, err
	}
	if costInput.ClusterID == "" {
		r.cvClientBuilder = cv.NewBuilder()
		err = GetClusterID(r, cost, costInput)
		if err != nil {
			log.Error(err, "Failed to obtain clusterID.")
			return ctrl.Result{}, err
		}
	}
	log.Info("Using the following inputs", "CostManagementInput", costInput)

	// Obtain credentials token/basic
	if costInput.Authentication == costmgmtv1alpha1.Token {
		// Get token from pull secret
		err = GetPullSecretToken(r, costInput)
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
	} else if costInput.AuthenticationSecretName != "" {
		// Get user and password from auth secret in namespace
		err = GetAuthSecret(r, costInput, req.NamespacedName)
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
	// Grab the Operator git commit and upload the status and input object with it
	commit, err := ioutil.ReadFile("commit")
	if err != nil {
		fmt.Println("File reading error", err)
		return ctrl.Result{}, err
	}
	cost.Status.OperatorCommit = strings.Replace(string(commit), "\n", "", -1)
	costInput.OperatorCommit = cost.Status.OperatorCommit
	err = r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
	}
	// Upload to c.rh.com
	var uploadStatus string
	var uploadTime string
	var body *bytes.Buffer
	var mw *multipart.Writer
	// Instead of looking for tarfiles here - we need to do what the old
	// operator did and create the tarfiles based on the CSV files and then get
	// a list of the tarfiles that are created
	files, err := ioutil.ReadDir("/tmp/cost-mgmt-operator-reports/upload")
	if err != nil {
		log.Error(err, "Could not read the directory")
	}
	if len(files) > 0 {
		log.Info("Pausing for " + fmt.Sprintf("%d", costInput.UploadWait) + " seconds before uploading.")
		time.Sleep(time.Duration(costInput.UploadWait) * time.Second)
	}
	for _, file := range files {
		log.Info("Uploading the following file: ")
		fmt.Println(file.Name())
		if strings.Contains(file.Name(), "tar.gz") {

			// grab the body and the multipart file header
			body, mw = GetBodyAndHeaders(r, "/tmp/cost-mgmt-operator-reports/"+file.Name())
			uploadStatus, uploadTime, err = Upload(r, costInput, "POST", costInput.IngressURL, body, mw)
			if err != nil {
				return ctrl.Result{}, err
			}
			if uploadStatus != "" {
				cost.Status.LastUploadStatus = uploadStatus
				costInput.LastUploadStatus = cost.Status.LastUploadStatus
				cost.Status.LastUploadTime = uploadTime
				costInput.LastUploadTime = cost.Status.LastUploadTime
				if strings.Contains(uploadStatus, "202") {
					cost.Status.LastSuccessfulUploadTime = uploadTime
					costInput.LastSuccessfulUploadTime = cost.Status.LastSuccessfulUploadTime
				}
				err = r.Status().Update(ctx, cost)
				if err != nil {
					log.Error(err, "Failed to update CostManagement Status")
				}
			}
		}
	}

	log.Info("Using the following inputs with creds", "CostManagementInput", costInput) // TODO remove after upload code works

	promConn, err := collector.GetPromConn(ctx, r.Client, r.Log)
	if err != nil {
		log.Error(err, "failed to get prometheus connection")
		cost.Status.Prometheus.PrometheusConnected = pointer.Bool(false)
		costInput.PrometheusConnected = *cost.Status.Prometheus.PrometheusConnected
		if err := r.Status().Update(ctx, cost); err != nil {
			log.Error(err, "failed to update CostManagement Status")
		}
	} else {
		cost.Status.Prometheus.PrometheusConnected = pointer.Bool(true)
		costInput.PrometheusConnected = *cost.Status.Prometheus.PrometheusConnected

		if cost.Status.Prometheus.LastQuerySuccessTime.Hour() != metav1.Now().Hour() {
			start := metav1.Now()
			cost.Status.Prometheus.LastQueryStartTime = start
			err = collector.DoQuery(promConn, r.Log)
			if err != nil {
				log.Error(err, "failed to query prometheus")
			} else {
				log.Info("prometheus queries completed")
				cost.Status.Prometheus.LastQuerySuccessTime = start
			}
		} else {
			log.Info("prometheus queries already complete for this hour")
		}

		if err := r.Status().Update(ctx, cost); err != nil {
			log.Error(err, "failed to update CostManagement Status")
		}
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
