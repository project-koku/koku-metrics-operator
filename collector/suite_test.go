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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

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

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx = context.Background()
var testSecretData = "this-is-the-data"
var testLogger = testutils.TestLogger{}

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Collector Suite",
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
			CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
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

func createPullSecretWithCert(namespace, prefix, dataKey string, data []byte) string {
	ctx := context.Background()
	name := prefix + "-" + testutils.RandomStringWithCharset(5, charset)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			dataKey:          data,
			"service-ca.crt": createCertificate("./test_files/service-ca.crt"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
	return name
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
				[]byte(testutils.RandomStringWithCharset(30, charset))),
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

func createCertificate(filename string) []byte {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IsCA: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	certOut, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	dat, err := ioutil.ReadFile(certOut.Name())
	if err != nil {
		log.Fatalf("Failed to read cert file: %v", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %v", err)
	}
	return dat
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}
