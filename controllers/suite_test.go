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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	configv1 "github.com/openshift/api/config/v1"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager
var testEnv *envtest.Environment
var useCluster bool

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	// Default to run locally
	var useClusterEnv string
	var ok bool
	var err error
	useCluster = false
	if useClusterEnv, ok = os.LookupEnv("USE_CLUSTER"); ok {
		useCluster, err = strconv.ParseBool(useClusterEnv)
	}

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster: &useCluster,
		CRDDirectoryPaths:  []string{filepath.Join("..", "config", "crd", "bases")},
	}
	// log.Info("\n\n\n\n LOOK HERE!!!! ")
	// log.Info(testEnv)
	cfg, err = testEnv.Start()
	// Expect(err).To(HaveOccurred())
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = costmgmtv1alpha1.AddToScheme(scheme.Scheme)
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

	if !useCluster {
		err = (&CostManagementReconciler{
			Client: k8sManager.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("CostManagementReconciler"),
			Scheme: scheme.Scheme,
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	clusterPrep()

	close(done)
}, 60)

func createNamespace(namespace string) {
	ctx := context.Background()
	instance := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
}

func createClusterVersion(clusterID string) {
	ctx := context.Background()
	instance := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID(clusterID),
		},
	}
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
}

func fakeDockerConfig() []byte {
	d, _ := json.Marshal(
		serializedAuthMap{
			Auths: map[string]serializedAuth{
				pullSecretAuthKey: serializedAuth{Auth: ".."},
			},
		})
	return d
}

func createPullSecret(namespace string, data []byte) {
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{
		pullSecretDataKey: data,
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func deletePullSecret(namespace string, data []byte) {
	ctx := context.Background()
	oldsecret := &corev1.Secret{Data: map[string][]byte{
		pullSecretDataKey: data,
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}

	Expect(k8sClient.Delete(ctx, oldsecret)).Should(Succeed())
}

func createBadPullSecret(namespace string) {
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createAuthSecret(namespace string) {
	ctx := context.Background()
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

func createBadAuthSecret(namespace string) {
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      badAuthSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func createBadAuthPassSecret(namespace string) {
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{
		authSecretUserKey: []byte("user1"),
	},
		ObjectMeta: metav1.ObjectMeta{
			Name:      badAuthPassSecretName,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func deleteClusterVersion() {
	ctx := context.Background()
	instance := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID(clusterID),
		},
	}
	Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
}

func clusterPrep() {
	if !useCluster {
		// Create operator namespace
		createNamespace(namespace)

		// Create auth secret in operator namespace
		createAuthSecret(namespace)

		// Create an empty auth secret
		createBadAuthSecret(namespace)
		createBadAuthPassSecret(namespace)

		// Create openshift config namespace and secret
		createNamespace(openShiftConfigNamespace)
		createPullSecret(openShiftConfigNamespace, fakeDockerConfig())

		// Create cluster version
		createClusterVersion(clusterID)
	}
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
