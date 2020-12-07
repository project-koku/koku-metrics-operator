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
package collector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/project-koku/koku-metrics-operator/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/deprecated/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx = context.Background()
var testSecretData = "this-is-the-data"
var testLogger = testutils.TestLogger{}

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	t := true
	if os.Getenv("TEST_USE_EXISTING_CLUSTER") == "true" {
		testEnv = &envtest.Environment{
			UseExistingCluster: &t,
		}
	} else {
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		}
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func createNamespace(namespace string) {
	ctx := context.Background()
	instance := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
}

func createPullSecret(namespace, prefix, dataKey string, data []byte) string {
	ctx := context.Background()
	name := prefix + "-" + testutils.RandomStringWithCharset(5, charset)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			dataKey: data,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
	return name
}

func createListOfRandomSecrets(num int, namespace string) []corev1.ObjectReference {
	Expect(num).Should(BeNumerically(">", 0))
	secrets := []corev1.ObjectReference{}
	for i := 0; i < num; i++ {
		secrets = append(secrets, corev1.ObjectReference{
			Name: createPullSecret(
				namespace,
				testutils.RandomStringWithCharset(20, charset),
				testutils.RandomStringWithCharset(8, charset),
				fakeEncodedData(testutils.RandomStringWithCharset(30, charset))),
		})
	}
	return secrets
}

func createServiceAccount(namespace, saName, fakeData string) *corev1.ServiceAccount {
	ctx := context.Background()
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
		},
	}
	Expect(k8sClient.Create(ctx, sa)).Should(Succeed())
	return sa
}

func addSecretsToSA(secrets []corev1.ObjectReference, sa *corev1.ServiceAccount) {
	sa.Secrets = secrets
	Expect(k8sClient.Update(ctx, sa)).Should(Succeed())
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func fakeEncodedData(data string) []byte {
	d, _ := json.Marshal(data)
	return d
}
