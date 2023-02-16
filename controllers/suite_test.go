//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controllers

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

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/testutils"
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
	ctx                context.Context
	cancel             context.CancelFunc
	useCluster         bool
	secretsPath        = ""
	emptyDirDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "koku-metrics-operator",
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
									Name:      "koku-metrics-operator-reports",
									MountPath: "/tmp/koku-metrics-operator-reports",
								}},
						},
					},
					Volumes: []corev1.Volume{{
						Name: "koku-metrics-operator-reports",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
				},
			},
		},
	}
	pvcDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "koku-metrics-operator",
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
									Name:      "koku-metrics-operator-reports",
									MountPath: "/tmp/koku-metrics-operator-reports",
								}},
						},
					},
					Volumes: []corev1.Volume{{
						Name: "koku-metrics-operator-reports",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "koku-metrics-operator-data"}}}},
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
			filepath.Join("..", "config", "crd", "bases"),
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
	metricsPort := 8090 + config.GinkgoConfig.ParallelNode
	flag.StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%d", metricsPort), "The address the metric endpoint binds to.")
	flag.Parse()

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: metricsAddr,
	})
	Expect(err).ToNot(HaveOccurred())

	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	if !useCluster {
		err = (&MetricsConfigReconciler{
			Client:                        k8sManager.GetClient(),
			Scheme:                        scheme.Scheme,
			Clientset:                     clientset,
			InCluster:                     true,
			disablePreviousDataCollection: true,
			overrideSecretPath:            true,
		}).SetupWithManager(k8sManager)
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

}, 60)

func createNamespace(ctx context.Context, namespace string) {
	instance := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
}

func createClusterVersion(ctx context.Context, clusterID string, channel string) {
	instance := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID(clusterID),
			Channel:   channel,
		},
	}
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
}

func fakeDockerConfig() []byte {
	d, _ := json.Marshal(
		serializedAuthMap{
			Auths: map[string]serializedAuth{pullSecretAuthKey: {Auth: ".."}},
		})
	return d
}

func createPullSecret(ctx context.Context, namespace string, data []byte) {
	secret := &corev1.Secret{Data: map[string][]byte{
		pullSecretDataKey: data,
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func deletePullSecret(ctx context.Context, namespace string, data []byte) {
	oldsecret := &corev1.Secret{Data: map[string][]byte{
		pullSecretDataKey: data,
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}

	Expect(k8sClient.Delete(ctx, oldsecret)).Should(Succeed())
}

func createBadPullSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createAuthSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{
		authSecretUserKey:     []byte("user1"),
		authSecretPasswordKey: []byte("password1"),
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      authSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}
func createMixedCaseAuthSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{
		"UserName": []byte("user1"),
		"PassWord": []byte("password1"),
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      authMixedCaseName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createBadAuthSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      badAuthSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createBadAuthPassSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{
		authSecretUserKey: []byte("user1"),
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      badAuthPassSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createBadAuthUserSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{Data: map[string][]byte{
		authSecretPasswordKey: []byte("password1"),
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      badAuthUserSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func deleteClusterVersion(ctx context.Context) {
	instance := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID(clusterID),
		},
	}
	Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
}

func createDeployment(ctx context.Context, deployment *appsv1.Deployment) {
	Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
}

func deleteDeployment(ctx context.Context, deployment *appsv1.Deployment) {
	Expect(k8sClient.Delete(ctx, deployment)).Should(Succeed())
}

func clusterPrep(ctx context.Context) {
	if !useCluster {
		// Create operator namespace
		createNamespace(ctx, namespace)

		// Create auth secret in operator namespace
		createAuthSecret(ctx, namespace)
		createMixedCaseAuthSecret(ctx, namespace)

		// Create an empty auth secret
		createBadAuthSecret(ctx, namespace)
		createBadAuthPassSecret(ctx, namespace)
		createBadAuthUserSecret(ctx, namespace)

		// Create openshift config namespace and secret
		createNamespace(ctx, openShiftConfigNamespace)
		createPullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())

		// Create cluster version
		createClusterVersion(ctx, clusterID, channel)

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

	validTS.Close()
	unauthorizedTS.Close()
})
