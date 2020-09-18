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
	"math/rand"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	cv "github.com/project-koku/korekuta-operator-go/clusterversion"
)

// CostManagementReconciler reconciles a CostManagement object
type CostManagementReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	cvClientBuilder cv.ClusterVersionBuilder
}

type CostManagementInput struct {
	ClusterID                string
	ValidateCert             bool
	IngressUrl               string
	AuthenticationSecretName string
	Authentication           costmgmtv1alpha1.AuthenticationType
	UploadWait               int64
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

// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-mgmt.openshift.io,resources=costmanagements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies;networks,verbs=get;list
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews;tokenreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list
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

	log.Info("Reconiling custom resource", "CostManagement", cost)
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

	return ctrl.Result{}, nil
}

func (r *CostManagementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costmgmtv1alpha1.CostManagement{}).
		Complete(r)
}
