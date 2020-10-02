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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	cv "github.com/project-koku/korekuta-operator-go/clusterversion"
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
	IngressUrl               string
	AuthenticationSecretName string
	Authentication           costmgmtv1alpha1.AuthenticationType
	UploadWait               int64
	BearerTokenString        string
	BasicAuthUser            string
	BasicAuthPassword        string
	LastUploadStatus         string
	LastUploadTime           string
	OperatorCommit           string
}

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}
type serializedAuth struct {
	Auth string `json:"auth"`
}

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

func ReflectSpec(r *CostManagementReconciler, cost *costmgmtv1alpha1.CostManagement, costInput *CostManagementInput) error {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "ReflectSpec")
	costInput.IngressUrl = StringReflectSpec(r, cost, &cost.Spec.IngressUrl, &cost.Status.IngressUrl, costmgmtv1alpha1.DefaultIngressUrl)
	costInput.AuthenticationSecretName = StringReflectSpec(r, cost, &cost.Spec.AuthenticationSecretName, &cost.Status.AuthenticationSecretName, "")
	// costInput.LastUploadStatus = StringReflectSpec(r, cost, &cost.Status.LastUploadStatus, "")

	if cost.Status.Authentication == "" || !reflect.DeepEqual(cost.Spec.Authentication, cost.Status.Authentication) {
		// If data is specified in the spec it should be used
		if cost.Spec.Authentication != "" {
			cost.Status.Authentication = cost.Spec.Authentication
		} else {
			cost.Status.Authentication = costmgmtv1alpha1.DefaultAuthenticationType
		}
	}
	costInput.Authentication = cost.Status.Authentication

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

	err := r.Status().Update(ctx, cost)
	if err != nil {
		log.Error(err, "Failed to update CostManagement Status")
		return err
	}
	return nil
}

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

func Upload(r *CostManagementReconciler, costInput *CostManagementInput) (string, string, error) {
	ctx := context.Background()
	log := r.Log.WithValues("costmanagement", "Upload")
	// log.Info("Inside of the upload function!")
	// Create the empty request
	req, err := http.NewRequest("POST", costInput.IngressUrl, nil)
	if err != nil {
		log.Error(err, "Could not send request")
	}
	// Create the header
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	// set the content and conetent type
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", "payload.tar.gz"))
	h.Set("Content-Type", "application/vnd.redhat.hccm.tar+tgz")
	fw, err := mw.CreatePart(h)
	req = req.WithContext(ctx)
	f, err := os.Open("payload.tar.gz")
	if err != nil {
		log.Info("error opening file %s", err)
	}
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		log.Error(err, "Could not send request")
	}
	mw.Close()
	req, err = http.NewRequest("POST", costInput.IngressUrl, buf)
	if err != nil {
		log.Error(err, "Could not send request")
	}

	// define the caCert
	// caCert, err := ioutil.ReadFile("ca-bundle.crt")
	// if err != nil {
	// 	log.Error(err, "An error Occurred")
	// }
	// caCertPool := x509.NewCertPool()
	// caCertPool.AppendCertsFromPEM(caCert)
	//
	// client := &http.Client{
	// 	Transport: &http.Transport{
	// 		TLSClientConfig: &tls.Config{
	// 			RootCAs: caCertPool,
	// 		},
	// 	},
	// }
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if costInput.Authentication == "basic" {
		log.Info("Uploading using basic authentication!")
		req.SetBasicAuth(costInput.BasicAuthUser, costInput.BasicAuthPassword)
	} else {
		log.Info("Uploading using token authentication")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", costInput.BearerTokenString))
		req.Header.Set("User-Agent", fmt.Sprintf("cost-mgmt-operator/%s cluster/%s", costInput.OperatorCommit, costInput.ClusterID))
	}
	for key, val := range req.Header {
		log.Info("Here is a header:")
		fmt.Println(key, val)
	}
	client := &http.Client{}
	// log.Info("Pausing for %s", costInput.UploadWait)
	// s := fmt.Sprintf("%+8d", costInput.UploadWait)
	// log.Info("Pausing for " + s)
	log.Info("Pausing for " + fmt.Sprintf("%d", costInput.UploadWait) + " seconds before uploading.")
	time.Sleep(time.Duration(costInput.UploadWait) * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
	}
	//log.Info(fmt.Sprintf("Request body: %q", req.Body))
	// requestID := resp.Header.Get("x-rh-insights-request-id")
	log.Info("Made it past the requestID!")
	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))
	// costInput.LastUploadStatus = http.StatusText(resp.StatusCode)
	//
	//
	// cost := &costmgmtv1alpha1.CostManagement{}
	// cost.Status.LastUploadStatus = fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	// costInput.LastUploadStatus = cost.Status.LastUploadStatus
	uploadStatus := fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	uploadTime := time.Now()
	// cost.Status.LastUploadTime = dt.String()
	// costInput.LastUploadTime = cost.Status.LastUploadTime
	//
	//
	// err = r.Status().Update(ctx, cost)
	// if err != nil {
	// 	log.Error(err, "Failed to update CostManagement Status")
	// }
	// if resp.StatusCode == http.StatusUnauthorized {
	// 	log.Info("gateway server %s returned 401, x-rh-insights-request-id=%s", resp.Request.URL, requestID)
	// 	// return authorizer.Error{Err: fmt.Errorf("your Red Hat account is not enabled for remote support or your token has expired")}
	// }
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "There was an error")
	}
	bodyString := string(bodyBytes)
	log.Info("The following is the response body:")
	log.Info(bodyString)

	return uploadStatus, uploadTime.Format("2006-01-02 15:04:05"), err
}

// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies;networks,verbs=get;list
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews;tokenreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=list;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,namespace=openshift-cost,resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets,verbs=create;delete;get;list;patch;update;watch

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
			cost.Status.AuthenticationCredentialsFound = pointer.Bool(false)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		} else {
			cost.Status.AuthenticationCredentialsFound = pointer.Bool(true)
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
			cost.Status.AuthenticationCredentialsFound = pointer.Bool(false)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		} else {
			cost.Status.AuthenticationCredentialsFound = pointer.Bool(true)
			err = r.Status().Update(ctx, cost)
			if err != nil {
				log.Error(err, "Failed to update CostManagement Status")
			}
		}
	} else {
		// No authentication secret name set when using basic auth
		cost.Status.AuthenticationCredentialsFound = pointer.Bool(false)
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
	uploadStatus, uploadTime, err = Upload(r, costInput)
	// Error encountered collecting authentication
	if err != nil {
		return ctrl.Result{}, err
	}
	if uploadStatus != "" {
		cost.Status.LastUploadStatus = uploadStatus
		costInput.LastUploadStatus = cost.Status.LastUploadStatus
		cost.Status.LastUploadTime = uploadTime
		costInput.LastUploadTime = cost.Status.LastUploadTime
		err = r.Status().Update(ctx, cost)
		if err != nil {
			log.Error(err, "Failed to update CostManagement Status")
		}
	}

	log.Info("Using the following inputs with creds", "CostManagementInput", costInput) // TODO remove after upload code works

	// Requeue for processing after 5 minutes
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func (r *CostManagementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costmgmtv1alpha1.CostManagement{}).
		Complete(r)
}
