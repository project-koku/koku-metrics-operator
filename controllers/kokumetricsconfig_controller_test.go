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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kokumetricscfgv1alpha1 "github.com/project-koku/koku-metrics-operator/api/v1alpha1"
	"github.com/project-koku/koku-metrics-operator/testutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	testLogger                  = testutils.TestLogger{}
	namespace                   = "koku-metrics-operator"
	namePrefix                  = "cost-test-local-"
	clusterID                   = "10e206d7-a11a-403e-b835-6cff14e98b23"
	sourceName                  = "cluster-test"
	authSecretName              = "cloud-dot-redhat"
	badAuthSecretName           = "baduserpass"
	badAuthPassSecretName       = "badpass"
	falseValue            bool  = false
	trueValue             bool  = true
	defaultUploadCycle    int64 = 360
	defaultCheckCycle     int64 = 1440
	defaultUploadWait     int64 = 0
	defaultAPIURL               = "https://not-the-real-cloud.redhat.com"
	instance                    = kokumetricscfgv1alpha1.KokuMetricsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
			Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
				AuthType: kokumetricscfgv1alpha1.Token,
			},
			Packaging: kokumetricscfgv1alpha1.PackagingSpec{
				MaxSize: 100,
			},
			Upload: kokumetricscfgv1alpha1.UploadSpec{
				UploadCycle:    &defaultUploadCycle,
				UploadToggle:   &trueValue,
				IngressAPIPath: "/api/ingress/v1/upload",
				ValidateCert:   &trueValue,
			},
			Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
				CreateSource:   &falseValue,
				SourcesAPIPath: "/api/sources/v1.0/",
				CheckCycle:     &defaultCheckCycle,
			},
			PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
				SkipTLSVerification: &trueValue,
				SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
			},
			APIURL: "https://not-the-real-cloud.redhat.com",
		},
	}
)

func Copy(mode os.FileMode, src, dst string) (os.FileInfo, error) {
	in, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return nil, err
	}
	info, err := out.Stat()
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(out.Name(), mode); err != nil {
		return nil, err
	}

	return info, out.Close()
}

func TestGetClientset(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	fmt.Printf("CURRENT DIR: %s\n", dir)

	ErrClientset := errors.New("test error")
	getClientsetTests := []struct {
		name        string
		config      string
		expectedErr error
	}{
		{name: "no config file", config: "", expectedErr: ErrClientset},
		{name: "fake config file", config: "test_files/kubeconfig", expectedErr: nil},
		// {name: "no decimal", input: 1234567890, want: "1234567890.000000"},
	}
	for _, tt := range getClientsetTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Setenv("KUBECONFIG", filepath.Join(dir, tt.config)); err != nil {
				t.Fatal("failed to set KUBECONFIG variable")
			}
			defer func() { os.Unsetenv("KUBECONFIG") }()
			got, err := GetClientset()
			if err == nil && tt.expectedErr != nil {
				t.Errorf("%s expected error but got %v", tt.name, err)
			}
			if err != nil && tt.expectedErr == nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.expectedErr == nil && got == nil {
				t.Errorf("%s result is unexpectedly nil", tt.name)
			}
		})
	}
}

func setup() error {
	type dirInfo struct {
		dirName  string
		files    []string
		dirMode  os.FileMode
		fileMode os.FileMode
	}
	testFiles := []string{"testFile.tar.gz"}
	dirInfoList := []dirInfo{
		{
			dirName:  "data",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "upload",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
	}
	// setup the initial testing directory
	testLogger.Info("setting up for controller tests")
	testingDir := "/tmp/koku-metrics-operator-reports/"
	if _, err := os.Stat(testingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testingDir, os.ModePerm); err != nil {
			return fmt.Errorf("could not create %s directory: %v", testingDir, err)
		}
	}
	for _, directory := range dirInfoList {
		reportPath := filepath.Join(testingDir, directory.dirName)
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			if err := os.Mkdir(reportPath, directory.dirMode); err != nil {
				return fmt.Errorf("could not create %s directory: %v", reportPath, err)
			}
			for _, reportFile := range directory.files {
				_, err := Copy(directory.fileMode, filepath.Join("test_files/", reportFile), filepath.Join(reportPath, reportFile))
				if err != nil {
					return fmt.Errorf("could not copy %s file: %v", reportFile, err)
				}
			}
		}
	}
	return nil
}

func shutdown() {
	testLogger.Info("tearing down for controller tests")
	os.RemoveAll("/tmp/koku-metrics-operator-reports/")
}

var _ = Describe("KokuMetricsConfigController", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that do not clean up leave resources behind.
		shutdown()
	})

	AfterEach(func() {
		shutdown()

	})

	Describe("KokuMetricsConfig CRD Handling", func() {
		Context("Process CRD resource", func() {
			It("should provide defaults for empty CRD case", func() {
				GitCommit = "1234567"

				instCopy := instance.DeepCopy()
				instCopy.ObjectMeta.Name = namePrefix + "emptycrd"
				Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

				fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

				// wait until the cluster ID is set
				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.ClusterID != ""
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
				Expect(fetched.Status.OperatorCommit).To(Equal(GitCommit))
				Expect(fetched.Status.Upload.UploadWait).NotTo(BeNil())
			})
		})
		It("upload set to false case", func() {

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "uploadfalse"
			instCopy.Spec.Upload.UploadToggle = &falseValue
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadToggle).To(Equal(&falseValue))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should find basic auth creds for good basic auth CRD case", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "basicauthgood"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())
			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should fail for missing basic auth token for bad basic auth CRD case", func() {
			badAuth := "bad-auth"
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "basicauthbad"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuth
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}
			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuth))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should reflect source name in status for source info CRD case", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "sourceinfo"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Source.SourceName = sourceName
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should reflect source error when attempting to create source", func() {

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "sourcecreate"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Source.SourceName = sourceName
			instCopy.Spec.Source.CreateSource = &trueValue
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
			Expect(fetched.Status.Source.SourceError).NotTo(BeNil())
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should fail due to bad basic auth secret", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "badauthsecret"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuthSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should fail due to missing pass in auth secret", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "badpass"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuthPassSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should fail due to missing auth secret name with basic set", func() {

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "missingname"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(""))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
		})
		It("should should fail due to deleted token secret", func() {
			deletePullSecret(openShiftConfigNamespace, fakeDockerConfig())
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "nopullsecret"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should should fail token secret wrong data", func() {
			createBadPullSecret(openShiftConfigNamespace)

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "nopulldata"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail bc of missing cluster version", func() {
			deleteClusterVersion()
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "missingcvfailure"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1alpha1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			time.Sleep(time.Second * 5)
			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(""))
		})
		It("should attempt upload due to tar.gz being present", func() {
			err := setup()
			Expect(err, nil)
			createClusterVersion(clusterID)
			deletePullSecret(openShiftConfigNamespace, fakeDockerConfig())
			createPullSecret(openShiftConfigNamespace, fakeDockerConfig())

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "attemptupload"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadError).NotTo(BeNil())
			Expect(fetched.Status.Upload.LastUploadStatus).NotTo(BeNil())
		})
		It("should check the last upload time in the upload status", func() {
			err := setup()
			Expect(err, nil)
			uploadTime := metav1.Now()
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "checkuploadstatus"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Status.Upload.LastSuccessfulUploadTime = uploadTime
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
	})
})
