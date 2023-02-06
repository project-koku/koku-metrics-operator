//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

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
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/collector"
	"github.com/project-koku/koku-metrics-operator/storage"
	"github.com/project-koku/koku-metrics-operator/testutils"
)

var (
	namespace                   = "koku-metrics-operator"
	namePrefix                  = "cost-test-local-"
	clusterID                   = "10e206d7-a11a-403e-b835-6cff14e98b23"
	channel                     = "4.8-stable"
	sourceName                  = "cluster-test"
	authSecretName              = "cloud-dot-redhat"
	authMixedCaseName           = "mixed-case"
	badAuthSecretName           = "baduserpass"
	badAuthPassSecretName       = "badpass"
	badAuthUserSecretName       = "baduser"
	falseValue            bool  = false
	trueValue             bool  = true
	defaultContextTimeout int64 = 120
	diffContextTimeout    int64 = 10
	defaultUploadCycle    int64 = 360
	defaultCheckCycle     int64 = 1440
	defaultUploadWait     int64 = 0
	defaultMaxReports     int64 = 1
	defaultAPIURL               = "https://not-the-real-cloud.redhat.com"
	instance                    = kokumetricscfgv1beta1.KokuMetricsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: kokumetricscfgv1beta1.KokuMetricsConfigSpec{
			Authentication: kokumetricscfgv1beta1.AuthenticationSpec{
				AuthType: kokumetricscfgv1beta1.Token,
			},
			Packaging: kokumetricscfgv1beta1.PackagingSpec{
				MaxSize:    100,
				MaxReports: defaultMaxReports,
			},
			Upload: kokumetricscfgv1beta1.UploadSpec{
				UploadCycle:    &defaultUploadCycle,
				UploadToggle:   &trueValue,
				IngressAPIPath: "/api/ingress/v1/upload",
				ValidateCert:   &trueValue,
			},
			Source: kokumetricscfgv1beta1.CloudDotRedHatSourceSpec{
				CreateSource:   &falseValue,
				SourcesAPIPath: "/api/sources/v1.0/",
				CheckCycle:     &defaultCheckCycle,
			},
			PrometheusConfig: kokumetricscfgv1beta1.PrometheusSpec{
				ContextTimeout:      &defaultContextTimeout,
				SkipTLSVerification: &trueValue,
				SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
			},
			APIURL: "https://not-the-real-cloud.redhat.com",
		},
	}
	airGappedInstance = kokumetricscfgv1beta1.KokuMetricsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: kokumetricscfgv1beta1.KokuMetricsConfigSpec{
			Authentication: kokumetricscfgv1beta1.AuthenticationSpec{
				AuthType: kokumetricscfgv1beta1.Token,
			},
			Packaging: kokumetricscfgv1beta1.PackagingSpec{
				MaxSize:    100,
				MaxReports: defaultMaxReports,
			},
			Upload: kokumetricscfgv1beta1.UploadSpec{
				UploadCycle:    &defaultUploadCycle,
				UploadToggle:   &falseValue,
				IngressAPIPath: "/api/ingress/v1/upload",
				ValidateCert:   &trueValue,
			},
			Source: kokumetricscfgv1beta1.CloudDotRedHatSourceSpec{
				CreateSource:   &falseValue,
				SourcesAPIPath: "/api/sources/v1.0/",
				CheckCycle:     &defaultCheckCycle,
			},
			PrometheusConfig: kokumetricscfgv1beta1.PrometheusSpec{
				ContextTimeout:      &diffContextTimeout,
				SkipTLSVerification: &trueValue,
				SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
			},
			APIURL: "https://not-the-real-cloud.redhat.com",
		},
	}
	differentPVC = &kokumetricscfgv1beta1.EmbeddedPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		EmbeddedObjectMetadata: kokumetricscfgv1beta1.EmbeddedObjectMetadata{
			Name: "a-different-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI),
				},
			},
		},
	}
)

func MockPromConnTester(promconn collector.PrometheusConnection) error { return nil }
func MockPromConnSetter(promcoll *collector.PrometheusCollector) error {
	promcoll.PromConn = mockPrometheusConnection{}
	return nil
}

type mockPrometheusConnection struct{}

func (m mockPrometheusConnection) QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error) {
	return model.Matrix{}, nil, nil
}

func (m mockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time) (model.Value, promv1.Warnings, error) {
	return model.Matrix{}, nil, nil
}

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

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := m.Run()
	os.Exit(code)
}

func TestGetClientset(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	ErrClientset := errors.New("test error")
	getClientsetTests := []struct {
		name        string
		config      string
		expectedErr error
	}{
		{name: "no config file", config: "", expectedErr: ErrClientset},
		{name: "fake config file", config: "test_files/kubeconfig", expectedErr: nil},
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

func TestConcatErrors(t *testing.T) {
	concatErrorsTests := []struct {
		name   string
		errors []error
		want   string
	}{
		{
			name:   "0 errors",
			errors: nil,
			want:   "",
		},
		{
			name:   "1 error",
			errors: []error{errors.New("error1")},
			want:   "error1",
		},
		{
			name:   "2 errors",
			errors: []error{errors.New("error1"), errors.New("error2")},
			want:   "error1\nerror2",
		},
		{
			name:   "3 errors",
			errors: []error{errors.New("error1"), errors.New("error2"), errors.New("error3")},
			want:   "error1\nerror2\nerror3",
		},
	}
	for _, tt := range concatErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			got := concatErrs(tt.errors...)
			if got != nil && got.Error() != tt.want {
				t.Errorf("%s\ngot: %v\nwant: %v\n", tt.name, got.Error(), tt.want)
			}
			if got == nil && tt.want != "" {
				t.Errorf("%s expected nil error, got: %T", tt.name, got)
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
	previousValidation = nil
	os.RemoveAll("/tmp/koku-metrics-operator-reports/")
}

var _ = Describe("KokuMetricsConfigController - CRD Handling", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()
	emptyDep1 := emptyDirDeployment.DeepCopy()
	emptyDep2 := emptyDirDeployment.DeepCopy()

	GitCommit = "1234567"
	promConnTester = MockPromConnTester
	promConnSetter = MockPromConnSetter

	BeforeEach(func() {
		// failed test runs that do not clean up leave resources behind.
		shutdown()
	})

	AfterEach(func() {
		shutdown()
	})

	Context("Process CRD resource - prior PVC mount", func() {
		/*
			All tests within this Context are only checking the functionality of mounting the PVC
			All other reconciler tests, post PVC mount, are performed in the following Context

				1. test default CR -> will create and mount deployment. Reconciler returns without changing anything, so test checks that the PVC exists in the deployment
				2. re-use deployment, create new CR to mimic a pod reboot. Check storage status
				3. re-use deployment, create new CR with specked PVC. Reconciler returns without changing anything. Test checks that PVC for deployment changed
				4. new deployment, create new CR with specked PVC. Reconciler returns without changing anything. Test checks that PVC for deployment matches specked claim
				5. repeat of 2 -> again, Check storage status
		*/
		It("should create and mount PVC for CR without PVC spec", func() {
			createDeployment(ctx, emptyDep1)

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "no-pvc-spec-1"
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			// wait until the deployment vol has changed
			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep1.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].EmptyDir == nil &&
					fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName == "koku-metrics-operator-data"
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, instCopy)).To(Succeed())
		})
		It("should not mount PVC for CR without PVC spec - pvc already mounted", func() {
			// reuse the old deployment
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "no-pvc-spec-2"
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the deployment vol has changed
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.ClusterVersion).To(Equal(channel))
			Expect(fetched.Status.Storage).ToNot(BeNil())
			Expect(fetched.Status.Storage.VolumeMounted).To(BeTrue())
			Expect(fetched.Status.Storage.VolumeMounted).To(BeTrue())
			Expect(fetched.Status.PersistentVolumeClaim.Name).To(Equal(storage.DefaultPVC.Name))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should mount PVC for CR with new PVC spec - pvc already mounted", func() {
			// reuse the old deployment
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "pvc-spec-3"
			instCopy.Spec.VolumeClaimTemplate = differentPVC
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			// wait until the deployment vol has changed
			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep1.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != "koku-metrics-operator-data"
			}, timeout, interval).Should(BeTrue())

			deleteDeployment(ctx, emptyDep1)
			Expect(k8sClient.Delete(ctx, instCopy)).To(Succeed())
		})
		It("should mount PVC for CR with new PVC spec - pvc already mounted", func() {
			createDeployment(ctx, emptyDep2)
			// reuse the old deployment
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "pvc-spec-4"
			instCopy.Spec.VolumeClaimTemplate = differentPVC
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			// wait until the deployment vol has changed
			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep2.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].EmptyDir == nil &&
					fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName == "a-different-pvc"
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, instCopy)).To(Succeed())
		})
		It("should not mount PVC for CR without PVC spec - pvc already mounted", func() {
			// reuse the old deployment
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "pvc-spec-5"
			instCopy.Spec.VolumeClaimTemplate = differentPVC
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the deployment vol has changed
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Storage).ToNot(BeNil())
			Expect(fetched.Status.Storage.VolumeMounted).To(BeTrue())
			Expect(fetched.Status.PersistentVolumeClaim.Name).To(Equal(differentPVC.Name))

			deleteDeployment(ctx, emptyDep2)
			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})

	Context("Process CRD resource - post PVC mount - disconnected cluster", func() {
		It("basic auth works fine", func() {
			createDeployment(ctx, pvcDeployment)

			instCopy := airGappedInstance.DeepCopy()
			instCopy.Spec.APIURL = unauthorizedTS.URL
			instCopy.ObjectMeta.Name = namePrefix + "default-cr-air-gapped-basic"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = "not-existent-secret"
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeNil())
			Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())

			Expect(fetched.Status.APIURL).To(Equal(unauthorizedTS.URL))

			Expect(fetched.Status.Prometheus.ContextTimeout).To(Equal(&diffContextTimeout))

			Expect(fetched.Status.Source.SourceDefined).To(BeNil())
			Expect(fetched.Status.Source.LastSourceCheckTime.IsZero()).To(BeTrue())
			Expect(fetched.Status.Source.SourceError).To(Equal(""))

			Expect(fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()).To(BeTrue())
			Expect(*fetched.Status.Upload.UploadToggle).To(BeFalse())

		})
		It("token auth works fine", func() {
			instCopy := airGappedInstance.DeepCopy()
			instCopy.Spec.APIURL = unauthorizedTS.URL
			instCopy.ObjectMeta.Name = namePrefix + "default-cr-air-gapped-token"
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeNil())

			Expect(fetched.Status.APIURL).To(Equal(unauthorizedTS.URL))

			Expect(fetched.Status.Prometheus.ContextTimeout).To(Equal(&diffContextTimeout))

			Expect(fetched.Status.Source.SourceDefined).To(BeNil())
			Expect(fetched.Status.Source.LastSourceCheckTime.IsZero()).To(BeTrue())
			Expect(fetched.Status.Source.SourceError).To(Equal(""))

			Expect(fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()).To(BeTrue())
			Expect(*fetched.Status.Upload.UploadToggle).To(BeFalse())

		})
	})

	Context("Process CRD resource - post PVC mount - connected cluster", func() {
		It("default CR works fine", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "default-cr"
			instCopy.Spec.APIURL = validTS.URL
			instCopy.Spec.Source.SourceName = "INSERT-SOURCE-NAME"
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
			Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.OperatorCommit).To(Equal(GitCommit))
			Expect(fetched.Status.Prometheus.ContextTimeout).To(Equal(&defaultContextTimeout))
			Expect(*fetched.Status.Source.SourceDefined).To(BeFalse())
			Expect(fetched.Status.Source.SourceError).ToNot(Equal(""))
			Expect(fetched.Status.Upload.UploadWait).NotTo(BeNil())

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("upload set to false case", func() {

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "uploadfalse"
			instCopy.Spec.Upload.UploadToggle = &falseValue
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeNil())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadToggle).To(Equal(&falseValue))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should find basic auth creds for good basic auth CRD case", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "basicauthgood"
			instCopy.Spec.APIURL = validTS.URL
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())
			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should find basic auth creds for good basic auth CRD case but fail because creds are wrong", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "basicauthgood-unauthorized"
			instCopy.Spec.APIURL = unauthorizedTS.URL
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())
			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(fetched.Status.APIURL).To(Equal(unauthorizedTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should find basic auth creds for mixedcase basic auth CRD case", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "mixed-case"
			instCopy.Spec.APIURL = validTS.URL
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authMixedCaseName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())
			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authMixedCaseName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail for missing basic auth token for bad basic auth CRD case", func() {
			badAuth := "bad-auth"
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "basicauthbad"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuth
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}
			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuth))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should reflect source name in status for source info CRD case", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "sourceinfo"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Source.SourceName = sourceName
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should reflect source error when attempting to create source", func() {

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "sourcecreate"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Source.SourceName = sourceName
			instCopy.Spec.Source.CreateSource = &trueValue
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
			Expect(fetched.Status.Source.SourceError).NotTo(BeNil())
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail due to bad basic auth secret", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "badauthsecret"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuthSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail due to missing pass in auth secret", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "badpass"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuthPassSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail due to missing user in auth secret", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "baduser"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = badAuthUserSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthUserSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail due to missing auth secret name with basic set", func() {
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "missingname"
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(""))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should should fail due to deleted token secret", func() {
			deletePullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "nopullsecret"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should should fail token secret wrong data", func() {
			createBadPullSecret(ctx, openShiftConfigNamespace)

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "nopulldata"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should fail bc of missing cluster version", func() {
			deleteClusterVersion(ctx)
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "missingcvfailure"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// check the CR was created ok
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.Storage.VolumeMounted
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(""))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should attempt upload due to tar.gz being present", func() {
			err := setup()
			Expect(err, nil)
			createClusterVersion(ctx, clusterID, channel)
			deletePullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())
			createPullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "attemptupload"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadError).NotTo(BeNil())
			Expect(fetched.Status.Upload.LastUploadStatus).NotTo(BeNil())
			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("tar.gz being present - upload attempt should 'succeed'", func() {
			err := setup()
			Expect(err, nil)
			deletePullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())
			createPullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "attemptuploadsuccess"
			instCopy.Spec.APIURL = validTS.URL
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadError).To(Equal(""))
			Expect(fetched.Status.Upload.LastUploadStatus).To(ContainSubstring("202"))
			Expect(fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()).To(BeFalse())
			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("tar.gz being present - basic auth upload attempt should fail because of bad auth", func() {
			err := setup()
			Expect(err, nil)
			deletePullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())
			createPullSecret(ctx, openShiftConfigNamespace, fakeDockerConfig())

			hourAgo := metav1.Now().Time.Add(-time.Hour)

			previousValidation = &previousAuthValidation{
				secretName: authSecretName,
				username:   "user1",
				password:   "password1",
				err:        nil,
				timestamp:  metav1.Time{Time: hourAgo},
			}

			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "attemptupload-unauthorized"
			instCopy.Spec.APIURL = unauthorizedTS.URL
			instCopy.Spec.Authentication.AuthType = kokumetricscfgv1beta1.Basic
			instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.Basic))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(unauthorizedTS.URL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadError).NotTo(Equal(""))
			Expect(fetched.Status.Upload.LastUploadStatus).To(ContainSubstring("401"))
			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
		It("should check the last upload time in the upload status", func() {
			err := setup()
			Expect(err, nil)
			uploadTime := metav1.Now()
			instCopy := instance.DeepCopy()
			instCopy.ObjectMeta.Name = namePrefix + "checkuploadstatus"
			instCopy.Spec.Upload.UploadWait = &defaultUploadWait
			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			fetched := &kokumetricscfgv1beta1.KokuMetricsConfig{}

			// wait until the cluster ID is set
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			fetched.Status.Upload.LastSuccessfulUploadTime = uploadTime
			Eventually(func() bool {
				_ = k8sClient.Status().Update(ctx, fetched)
				return fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()
			}, timeout, interval).ShouldNot(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1beta1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})
})
