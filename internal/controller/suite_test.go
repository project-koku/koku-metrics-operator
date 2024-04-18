//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controller

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                *rest.Config
	clientset          *kubernetes.Clientset
	k8sClient          client.Client
	k8sManager         ctrl.Manager
	testEnv            *envtest.Environment
	defaultReconciler  *MetricsConfigReconciler
	ctx                context.Context
	cancel             context.CancelFunc
	useCluster         bool
	secretsPath        = ""
	emptyDirDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeMountName,
									MountPath: dirconfig.MountPath,
								}},
						},
					},
					Volumes: []corev1.Volume{{
						Name: volumeMountName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
				},
			},
		},
	}
	pvcDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeMountName,
									MountPath: dirconfig.MountPath,
								}},
						},
					},
					Volumes: []corev1.Volume{{
						Name: volumeMountName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: volumeClaimName}}}},
				},
			},
		},
	}
	differentPVC = &metricscfgv1beta1.EmbeddedPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		EmbeddedObjectMetadata: metricscfgv1beta1.EmbeddedObjectMetadata{
			Name: "a-different-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI),
				},
			},
		},
	}

	validTS        *httptest.Server
	unauthorizedTS *httptest.Server
)

func int32Ptr(i int32) *int32 { return &i }

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	validTS = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ingress") {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintln(w, "Upload Accepted")
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, client")
		}
	}))
	unauthorizedTS = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	logf.SetLogger(testutils.ZapLogger(true))
	ctx, cancel = context.WithCancel(context.Background())

	// Default to run locally
	var useClusterEnv string
	var ok bool
	var err error
	useCluster = false
	if useClusterEnv, ok = os.LookupEnv("USE_CLUSTER"); ok {
		useCluster, err = strconv.ParseBool(useClusterEnv)
		Expect(err).ToNot(HaveOccurred())
	}

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster: &useCluster,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("test_files", "crds"),
		},
	}
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = metricscfgv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = operatorsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = configv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	// make the metrics listen address different for each parallel thread to avoid clashes when running with -p
	var metricsAddr string
	metricsPort := 8090 + GinkgoParallelProcess()
	flag.StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%d", metricsPort), "The address the metric endpoint binds to.")
	flag.Parse()

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: metricsAddr},
	})
	Expect(err).ToNot(HaveOccurred())

	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	if !useCluster {
		defaultReconciler = &MetricsConfigReconciler{
			Client:             k8sManager.GetClient(),
			Scheme:             scheme.Scheme,
			Clientset:          clientset,
			InCluster:          true,
			overrideSecretPath: true,
		}
		err := (defaultReconciler).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	}

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	clusterPrep(ctx)

})

type ReconcilerOption func(f *MetricsConfigReconciler)

func WithSecretOverride(overrideSecretPath bool) ReconcilerOption {
	return func(r *MetricsConfigReconciler) {
		r.overrideSecretPath = overrideSecretPath
	}
}

func resetReconciler(opts ...ReconcilerOption) {
	defaultReconciler.promCollector = nil
	defaultReconciler.overrideSecretPath = true
	for _, opt := range opts {
		opt(defaultReconciler)
	}
}

func createNamespace(ctx context.Context, namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	createObject(ctx, ns)
}

func createClusterVersion(ctx context.Context) {
	key := types.NamespacedName{Name: "version"}
	cv := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID(clusterID),
			Channel:   channel,
		},
	}
	createObject(ctx, cv)
}

func deleteClusterVersion(ctx context.Context) {
	cv := &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version"}}
	deleteObject(ctx, cv)
}

func fakeDockerConfig() map[string][]byte {
	d, _ := json.Marshal(
		serializedAuthMap{
			Auths: map[string]serializedAuth{pullSecretAuthKey: {Auth: ".."}},
		})
	return map[string][]byte{pullSecretDataKey: d}
}

func createSecret(ctx context.Context, name, namespace string, data map[string][]byte) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	createObject(ctx, secret)
}

func deleteSecret(ctx context.Context, name, namespace string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	deleteObject(ctx, secret)
}

func createPullSecret(ctx context.Context, data map[string][]byte) {
	createSecret(ctx, pullSecretName, openShiftConfigNamespace, data)
}

func deletePullSecret(ctx context.Context) {
	deleteSecret(ctx, pullSecretName, openShiftConfigNamespace)
}

func createAuthSecret(ctx context.Context) {
	secretData := map[string][]byte{
		authSecretUserKey:     []byte("user1"),
		authSecretPasswordKey: []byte("password1"),
	}
	createSecret(ctx, authSecretName, namespace, secretData)
}

func deleteAuthSecret(ctx context.Context) {
	deleteSecret(ctx, authSecretName, namespace)
}

func createServiceAccountSecret(ctx context.Context) {
	secretData := map[string][]byte{
		"client_id":     []byte("mockClientID"),
		"client_secret": []byte("mockclientSecret"),
	}

	createSecret(ctx, serviceAccountSecretName, namespace, secretData)
}

func deleteServiceAccountSecret(ctx context.Context) {
	deleteSecret(ctx, serviceAccountSecretName, namespace)
}

func replaceAuthSecretData(ctx context.Context, data map[string][]byte) {
	key := types.NamespacedName{
		Name:      authSecretName,
		Namespace: namespace,
	}
	secret := &corev1.Secret{}
	Expect(k8sClient.Get(ctx, key, secret)).Should(Succeed())

	secret.Data = data
	Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

	secret = &corev1.Secret{}
	Expect(k8sClient.Get(ctx, key, secret)).Should(Succeed())
	Expect(secret.Data).To(BeComparableTo(data, cmpopts.EquateEmpty()))
}

func createObject(ctx context.Context, obj client.Object) {
	key := client.ObjectKeyFromObject(obj)
	log.Info("CREATING OBJECT", "object", key)
	Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
	log.Info("CREATED OBJECT", "object", key)
}

func deleteObject(ctx context.Context, obj client.Object) {
	key := client.ObjectKeyFromObject(obj)
	log.Info("DELETING OBJECT", "object", key)
	Expect(k8sClient.Delete(ctx, obj)).Should(Or(Succeed(), Satisfy(errors.IsNotFound)))
	Eventually(func() bool { return errors.IsNotFound(k8sClient.Get(ctx, key, obj)) }, 60, 1).Should(BeTrue())
	log.Info("DELETED OBJECT", "object", key)
}

func ensureObjectExists(ctx context.Context, key types.NamespacedName, obj client.Object) {
	if err := k8sClient.Get(ctx, key, obj); errors.IsNotFound(err) {
		createObject(ctx, obj)
	}
}

func setupRequired(ctx context.Context) {
	createClusterVersion(ctx)
	createPullSecret(ctx, fakeDockerConfig())
	createAuthSecret(ctx)
	createServiceAccountSecret(ctx)
}

func tearDownRequired(ctx context.Context) {
	deleteClusterVersion(ctx)
	deletePullSecret(ctx)
	deleteAuthSecret(ctx)
	deleteServiceAccountSecret(ctx)
}

func clusterPrep(ctx context.Context) {
	if !useCluster {
		// Create operator namespace
		createNamespace(ctx, namespace)
		createNamespace(ctx, "openshift-monitoring")
		createNamespace(ctx, openShiftConfigNamespace)

		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		err = os.Setenv("SECRET_ABSPATH", filepath.Join(cwd, secretsPath))
		Expect(err).ToNot(HaveOccurred())

		testutils.CreateCertificate(secretsPath, "service-ca.crt")
		testutils.CreateToken(secretsPath, "token")
	}
}

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())

	os.Remove(filepath.Join(secretsPath, "token"))
	os.Remove(filepath.Join(secretsPath, "service-ca.crt"))
	os.RemoveAll(filepath.Join(secretsPath, "tmp"))

	validTS.Close()
	unauthorizedTS.Close()
})
