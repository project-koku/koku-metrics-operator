//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/collector"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/mocks"
	"github.com/project-koku/koku-metrics-operator/internal/storage"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var (
	namespace       = fmt.Sprintf("%s-metrics-operator", metricscfgv1beta1.NamePrefix)
	deploymentName  = fmt.Sprintf("%s-metrics-operator", metricscfgv1beta1.NamePrefix)
	volumeMountName = fmt.Sprintf("%s-metrics-operator-reports", metricscfgv1beta1.NamePrefix)
	volumeClaimName = fmt.Sprintf("%s-metrics-operator-data", metricscfgv1beta1.NamePrefix)

	testObjectNamePrefix           = "cost-test-local"
	clusterID                      = "10e206d7-a11a-403e-b835-6cff14e98b23"
	channel                        = "4.8-stable"
	sourceName                     = "cluster-test"
	authSecretName                 = "basic-auth-secret"
	serviceAccountSecretName       = "sa-auth-secret"
	falseValue               bool  = false
	trueValue                bool  = true
	defaultContextTimeout    int64 = 120
	diffContextTimeout       int64 = 10
	defaultUploadCycle       int64 = 360
	defaultCheckCycle        int64 = 1440
	defaultUploadWait        int64 = 0
	defaultMaxReports        int64 = 1
	defaultAPIURL                  = "https://not-the-real-console.redhat.com"
	testingDir                     = dirconfig.MountPath
)

type mockPrometheusConnection struct{}

func (m *mockPrometheusConnection) QueryRange(ctx context.Context, query string, r promv1.Range, opts ...promv1.Option) (model.Value, promv1.Warnings, error) {
	return model.Matrix{}, nil, nil
}

func (m *mockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time, opts ...promv1.Option) (model.Value, promv1.Warnings, error) {
	return model.Vector{}, nil, nil
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

func TestIsQueryNeeded(t *testing.T) {
	isQueryNeededTests := []struct {
		name  string
		time  time.Time
		count []int
		want  bool
	}{
		{
			name:  "time not in tracker",
			time:  time.Now(),
			count: []int{-1},
			want:  true,
		},
		{
			name:  "time in tracker, retry required",
			time:  time.Now(),
			count: []int{0, 1, 2, 3, 4},
			want:  true,
		},
		{
			name:  "time in tracker, retry count exceeded",
			time:  time.Now(),
			count: []int{5, 6, 7, 8, 9},
			want:  false,
		},
	}
	for _, tt := range isQueryNeededTests {
		for _, count := range tt.count {
			if count > -1 {
				retryTracker[tt.time] = count
			}
			t.Run(tt.name, func(t *testing.T) {
				got := isQueryNeeded(tt.time)
				if got != tt.want {
					t.Errorf("%s\ngot: %v\nwant: %v\n", tt.name, got, tt.want)
				}
			})
		}
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
	os.RemoveAll(testingDir)
}

var _ = Describe("MetricsConfigController - CRD Handling", Ordered, func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	var (
		r *MetricsConfigReconciler

		mockCtrl  *gomock.Controller
		mockpconn *mocks.MockPrometheusConnection

		testConfigMap *corev1.ConfigMap
		instCopy      *metricscfgv1beta1.MetricsConfig
		testPVC       *corev1.PersistentVolumeClaim
		checkPVC      bool = true
	)

	ctx := context.Background()
	emptyDep1 := emptyDirDeployment.DeepCopy()
	emptyDep2 := emptyDirDeployment.DeepCopy()

	// override MaxRetries to reduce testing time
	collector.MaxRetries = 1

	BeforeEach(func() {

		GitCommit = "1234567"

		setupRequired(ctx)
		retentionPeriod = time.Duration(0)

		promConnTester = func(promcoll *collector.PrometheusCollector) error { return nil }
		promConnSetter = func(promcoll *collector.PrometheusCollector) error {
			promcoll.PromConn = &mockPrometheusConnection{}
			return nil
		}
	})

	JustBeforeEach(func() {
		// ensure all tests use the correct time.Now()
		now = time.Now

		instCopy = &metricscfgv1beta1.MetricsConfig{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      testObjectNamePrefix,
			},
			Spec: metricscfgv1beta1.MetricsConfigSpec{
				Authentication: metricscfgv1beta1.AuthenticationSpec{
					AuthType: metricscfgv1beta1.Token,
				},
				Packaging: metricscfgv1beta1.PackagingSpec{
					MaxSize:    100,
					MaxReports: defaultMaxReports,
				},
				Upload: metricscfgv1beta1.UploadSpec{
					UploadCycle:    &defaultUploadCycle,
					UploadToggle:   &trueValue,
					UploadWait:     &defaultUploadWait,
					IngressAPIPath: "/api/ingress/v1/upload",
					ValidateCert:   &trueValue,
				},
				Source: metricscfgv1beta1.CloudDotRedHatSourceSpec{
					CreateSource:   &falseValue,
					SourcesAPIPath: "/api/sources/v1.0/",
					CheckCycle:     &defaultCheckCycle,
				},
				PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
					CollectPreviousData: &falseDef,
					ContextTimeout:      &defaultContextTimeout,
					SkipTLSVerification: &trueValue,
					SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
				},
				APIURL: "https://not-the-real-console.redhat.com",
			},
		}
		testConfigMap = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      monitoringMeta.Name,
				Namespace: monitoringMeta.Namespace,
			},
		}

		if checkPVC {
			testPVC = storage.MakeVolumeClaimTemplate(storage.DefaultPVC, namespace)

			ensureObjectExists(ctx, client.ObjectKeyFromObject(testPVC), testPVC)
			ensureObjectExists(ctx, client.ObjectKeyFromObject(pvcDeployment), pvcDeployment)
		}

	})

	JustAfterEach(func() {
		deleteObject(ctx, instCopy)
		deleteObject(ctx, testConfigMap)
	})

	AfterEach(func() {
		shutdown()

		tearDownRequired(ctx)
	})

	Context("Process CRD resource - prior PVC mount", func() {
		BeforeEach(func() {
			checkPVC = false
		})
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
			createObject(ctx, emptyDep1)

			Expect(k8sClient.Create(ctx, instCopy)).Should(Succeed())

			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep1.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].EmptyDir == nil &&
					fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName == volumeClaimName
			}, timeout, interval).Should(BeTrue())
		})
		It("should not mount PVC for CR without PVC spec - pvc already mounted", func() {
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.ClusterVersion).To(Equal(channel))
			Expect(fetched.Status.Storage).ToNot(BeNil())
			Expect(fetched.Status.Storage.VolumeMounted).To(BeTrue())
			Expect(fetched.Status.PersistentVolumeClaim.Name).To(Equal(storage.DefaultPVC.Name))
		})
		It("should mount PVC for CR with new PVC spec - pvc already mounted", func() {
			instCopy.Spec.VolumeClaimTemplate = differentPVC
			createObject(ctx, instCopy)

			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep1.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != volumeClaimName
			}, timeout, interval).Should(BeTrue())

			deleteObject(ctx, emptyDep1)
		})
		It("should mount PVC for CR with new PVC spec - pvc already mounted", func() {
			createObject(ctx, emptyDep2)

			instCopy.Spec.VolumeClaimTemplate = differentPVC
			createObject(ctx, instCopy)

			Eventually(func() bool {
				fetched := &appsv1.Deployment{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: emptyDep2.Name, Namespace: namespace}, fetched)
				return fetched.Spec.Template.Spec.Volumes[0].EmptyDir == nil &&
					fetched.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName == "a-different-pvc"
			}, timeout, interval).Should(BeTrue())
		})
		It("should not mount PVC for CR without PVC spec - pvc already mounted", func() {
			instCopy.Spec.VolumeClaimTemplate = differentPVC
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Storage).ToNot(BeNil())
			Expect(fetched.Status.Storage.VolumeMounted).To(BeTrue())
			Expect(fetched.Status.PersistentVolumeClaim.Name).To(Equal(differentPVC.Name))

			deleteObject(ctx, emptyDep2)
		})
	})

	Context("Process CRD resource - post PVC mount", func() {

		BeforeEach(func() {
			checkPVC = true
		})

		When("cluster is disconnected", func() {
			JustBeforeEach(func() {
				instCopy.Spec.Upload.UploadToggle = &falseValue
				instCopy.Spec.PrometheusConfig.ContextTimeout = &diffContextTimeout
			})
			It("basic auth works fine", func() {
				instCopy.Spec.APIURL = unauthorizedTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = "not-existent-secret"
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.ClusterID != ""
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
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
				instCopy.Spec.APIURL = unauthorizedTS.URL
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.ClusterID != ""
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
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

		When("cluster is connected", func() {
			BeforeEach(func() {
				checkPVC = true
			})
			It("default CR works fine", func() {
				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Source.SourceName = ""
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
				Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
				Expect(fetched.Status.OperatorCommit).To(Equal(GitCommit))
				Expect(fetched.Status.Prometheus.ContextTimeout).To(Equal(&defaultContextTimeout))
				Expect(*fetched.Status.Source.SourceDefined).To(BeFalse())
				Expect(fetched.Status.Source.SourceName).To(Equal(clusterID))
				Expect(fetched.Status.Source.SourceError).ToNot(Equal(""))
				Expect(fetched.Status.Upload.UploadWait).ToNot(BeNil())
			})
			It("upload set to false case", func() {
				instCopy.Spec.Upload.UploadToggle = &falseValue
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.ClusterID != ""
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Upload.UploadToggle).To(Equal(&falseValue))
				Expect(fetched.Status.Upload.UploadWait).To(Equal(&defaultUploadWait))
			})
			It("should find service account auth creds for good service account auth CRD case", func() {
				// Create a valid service account secret
				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Authentication.TokenURL = validTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.ServiceAccount
				instCopy.Spec.Authentication.AuthenticationSecretName = serviceAccountSecretName

				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.ServiceAccount))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(serviceAccountSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
				Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})

			It("should handle missing service account auth secret", func() {
				deleteServiceAccountSecret(ctx)

				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Authentication.TokenURL = validTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.ServiceAccount

				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.ServiceAccount))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(BeEmpty())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(BeEmpty())
				Expect(fetched.Status.Authentication.AuthErrorMessage).To(ContainSubstring("no authentication secret name set when using service-account auth"))
			})

			It("should handle missing required fields in service account auth creds", func() {

				// Create secret missing 'clientid' to make it invalid.
				secretData := map[string][]byte{
					"client_secret": []byte("mockClientSecret"),
				}
				secretName := "invalid-sa-secret"
				createSecret(ctx, secretName, namespace, secretData)

				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Authentication.TokenURL = validTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.ServiceAccount
				instCopy.Spec.Authentication.AuthenticationSecretName = secretName

				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.ServiceAccount))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(secretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).To(ContainSubstring("service account secret not found with expected client_id data"))
			})

			It("should find basic auth creds for good basic auth CRD case", func() {
				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeTrue())
				Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
			It("should find basic auth creds for good basic auth CRD case but fail because creds are wrong", func() {
				instCopy.Spec.APIURL = unauthorizedTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
			})
			It("should find basic auth creds for mixedcase basic auth CRD case", func() {
				mixedCaseData := map[string][]byte{
					"UserName": []byte("user1"),
					"PassWord": []byte("password1"),
				}
				replaceAuthSecretData(ctx, mixedCaseData)

				instCopy.Spec.APIURL = validTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeTrue())
			})
			It("should fail for missing basic auth token for bad basic auth CRD case", func() {
				deleteAuthSecret(ctx)

				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			})
			It("should reflect source name in status for source info CRD case", func() {
				instCopy.Spec.Source.SourceName = sourceName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
				Expect(fetched.Status.Source.LastSourceCheckTime.IsZero()).To(BeFalse())
			})
			It("should reflect source error when attempting to create source", func() {
				instCopy.Spec.Source.SourceName = sourceName
				instCopy.Spec.Source.CreateSource = &trueValue
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
				Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
				Expect(fetched.Status.Source.SourceError).ToNot(BeNil())
			})
			It("should fail due to bad basic auth secret", func() {
				replaceAuthSecretData(ctx, map[string][]byte{})

				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
				Expect(fetched.Status.APIURL).To(Equal(defaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
			It("should fail due to missing pass in auth secret", func() {
				replaceAuthSecretData(ctx, map[string][]byte{authSecretUserKey: []byte("user1")})

				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			})
			It("should fail due to missing user in auth secret", func() {
				replaceAuthSecretData(ctx, map[string][]byte{authSecretPasswordKey: []byte("password1")})

				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			})
			It("should fail due to missing auth secret name with basic set", func() {
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(""))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
			})
			It("should should fail due to deleted token secret", func() {
				deletePullSecret(ctx)

				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
			})
			It("should should fail token secret wrong data", func() {
				deletePullSecret(ctx)
				createPullSecret(ctx, map[string][]byte{}) // create a bad pullsecret

				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
			})
			It("should fail bc of missing cluster version", func() {
				deleteClusterVersion(ctx)

				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Storage.VolumeMounted
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.ClusterID).To(Equal(""))
			})
			It("should attempt upload due to tar.gz being present", func() {
				Expect(setup()).Should(Succeed())
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Upload.UploadError).ToNot(BeNil())
				Expect(fetched.Status.Upload.LastUploadStatus).ToNot(BeNil())
			})
			It("tar.gz being present - upload attempt should 'succeed'", func() {
				Expect(setup()).Should(Succeed())
				instCopy.Spec.APIURL = validTS.URL
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.APIURL).To(Equal(validTS.URL))
				Expect(fetched.Status.Upload.UploadError).To(Equal(""))
				Expect(fetched.Status.Upload.LastUploadStatus).To(ContainSubstring("202"))
				Expect(fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()).To(BeFalse())
			})
			It("tar.gz being present - basic auth upload attempt should fail because of bad auth", func() {
				Expect(setup()).Should(Succeed())
				hourAgo := metav1.Now().Time.Add(-time.Hour)

				previousValidation = &previousAuthValidation{
					secretName: authSecretName,
					username:   "user1",
					password:   "password1",
					err:        nil,
					timestamp:  metav1.Time{Time: hourAgo},
				}

				instCopy.Spec.APIURL = unauthorizedTS.URL
				instCopy.Spec.Authentication.AuthType = metricscfgv1beta1.Basic
				instCopy.Spec.Authentication.AuthenticationSecretName = authSecretName
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.Basic))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.Authentication.AuthErrorMessage).ToNot(Equal(""))
				Expect(*fetched.Status.Authentication.ValidBasicAuth).To(BeFalse())
				Expect(fetched.Status.APIURL).To(Equal(unauthorizedTS.URL))
				Expect(fetched.Status.Upload.UploadError).ToNot(Equal(""))
				Expect(fetched.Status.Upload.LastUploadStatus).To(ContainSubstring("401"))
			})
			It("should check the last upload time in the upload status", func() {
				Expect(setup()).Should(Succeed())
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.ClusterID != ""
				}, timeout, interval).Should(BeTrue())

				fetched.Status.Upload.LastSuccessfulUploadTime = metav1.Now()
				Eventually(func() bool {
					_ = k8sClient.Status().Update(ctx, fetched)
					return fetched.Status.Upload.LastSuccessfulUploadTime.IsZero()
				}, timeout, interval).Should(BeFalse())

				refetched := &metricscfgv1beta1.MetricsConfig{}
				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, refetched)
					return refetched.Status.Upload.LastSuccessfulUploadTime.IsZero()
				}, timeout, interval).Should(BeFalse())
			})
			It("old default url - should update status to new default url", func() {
				Expect(setup()).Should(Succeed())
				instCopy.Spec.APIURL = metricscfgv1beta1.OldDefaultAPIURL
				instCopy.Spec.Source.SourceName = "OLD-API-URL"
				createObject(ctx, instCopy)

				fetched := &metricscfgv1beta1.MetricsConfig{}

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
					return fetched.Status.Authentication.AuthenticationCredentialsFound != nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(metricscfgv1beta1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.Authentication.ValidBasicAuth).To(BeNil())
				Expect(fetched.Status.APIURL).To(Equal(metricscfgv1beta1.DefaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
		})
	})

	Context("set the correct retention period for data gather on CR creation", func() {

		BeforeEach(func() {
			r = &MetricsConfigReconciler{Client: k8sClient, apiReader: k8sManager.GetAPIReader()}
			Expect(retentionPeriod).To(Equal(time.Duration(0)))
		})
		It("configMap does not exist - uses 14 days", func() {
			setRetentionPeriod(ctx, r)
			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("no configMap is specified - uses 14 days", func() {
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with empty config.yaml - uses 14 days", func() {
			testConfigMap.Data = map[string]string{"config.yaml": ""}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with config.yaml without retention period - uses 14 days", func() {
			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  not-retention-period-string: 90d"}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with mangled config.yaml - uses 14 days", func() {
			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s\n  not-retention-period-string: 90d"}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with config.yaml with malformed retention period - uses 14 days", func() {
			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 90"}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(fourteenDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with config.yaml with valid retention period", func() {
			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 81d"}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(time.Duration(81 * 24 * time.Hour)))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})
		It("configMap is specified with config.yaml with valid retention period greater than 90d", func() {
			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 91d"}
			createObject(ctx, testConfigMap)

			setRetentionPeriod(ctx, r)

			Expect(retentionPeriod).To(Equal(ninetyDayDuration))
			Expect(retentionPeriod).ToNot(Equal(time.Duration(0)))
		})

		It("check the start time on new CR creation - previous data collection set to true", func() {
			// cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
			// *cr.Spec.PrometheusConfig.CollectPreviousData &&
			// cr.Status.Prometheus.LastQuerySuccessTime.IsZero()
			original := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)

			cr := &metricscfgv1beta1.MetricsConfig{
				Spec: metricscfgv1beta1.MetricsConfigSpec{
					PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
						CollectPreviousData: &trueDef,
					},
				},
			}

			got, _ := getTimeRange(ctx, r, cr)
			Expect(got).ToNot(Equal(original))
			Expect(got).To(Equal(original.Add(-fourteenDayDuration).Truncate(24 * time.Hour)))
		})

		It("check the start time on old CR - previous data collection set to true", func() {
			// cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
			// *cr.Spec.PrometheusConfig.CollectPreviousData &&
			// cr.Status.Prometheus.LastQuerySuccessTime.IsZero()
			original := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)

			cr := &metricscfgv1beta1.MetricsConfig{
				Spec: metricscfgv1beta1.MetricsConfigSpec{
					PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
						CollectPreviousData: &trueDef,
					},
				},
				Status: metricscfgv1beta1.MetricsConfigStatus{
					Prometheus: metricscfgv1beta1.PrometheusStatus{
						LastQuerySuccessTime: metav1.Now(),
					},
				},
			}

			got, _ := getTimeRange(ctx, r, cr)
			Expect(got).To(Equal(original))
			Expect(got).ToNot(Equal(original.Add(-fourteenDayDuration)))
		})

		It("check the start time on old CR - failed query more than hour old", func() {
			// cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
			// *cr.Spec.PrometheusConfig.CollectPreviousData &&
			// cr.Status.Prometheus.LastQuerySuccessTime.IsZero()
			original := time.Now().UTC().Truncate(time.Hour).Add(-2 * time.Hour)

			cr := &metricscfgv1beta1.MetricsConfig{
				Spec: metricscfgv1beta1.MetricsConfigSpec{
					PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
						CollectPreviousData: &falseDef,
					},
				},
				Status: metricscfgv1beta1.MetricsConfigStatus{
					Prometheus: metricscfgv1beta1.PrometheusStatus{
						LastQuerySuccessTime: metav1.Time{Time: original.Add(-1 * time.Hour)},
					},
				},
			}

			got, _ := getTimeRange(ctx, r, cr)
			Expect(got).To(Equal(original))
			Expect(got).ToNot(Equal(original.Add(-fourteenDayDuration)))
		})

		It("check the start time on old CR - failed query more than retention period old", func() {
			// cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
			// *cr.Spec.PrometheusConfig.CollectPreviousData &&
			// cr.Status.Prometheus.LastQuerySuccessTime.IsZero()
			thirtyDaysOld := time.Now().UTC().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
			expected := now().UTC().Add(-1 * fourteenDayDuration).Truncate(24 * time.Hour)

			cr := &metricscfgv1beta1.MetricsConfig{
				Spec: metricscfgv1beta1.MetricsConfigSpec{
					PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
						CollectPreviousData: &falseDef,
					},
				},
				Status: metricscfgv1beta1.MetricsConfigStatus{
					Prometheus: metricscfgv1beta1.PrometheusStatus{
						LastQuerySuccessTime: metav1.Time{Time: thirtyDaysOld},
					},
				},
			}

			got, _ := getTimeRange(ctx, r, cr)
			Expect(got).To(Equal(expected))
		})

		It("check the start time on new CR - previous data collection set to false", func() {
			// cr.Spec.PrometheusConfig.CollectPreviousData != nil &&
			// *cr.Spec.PrometheusConfig.CollectPreviousData &&
			// cr.Status.Prometheus.LastQuerySuccessTime.IsZero()
			original := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)

			cr := &metricscfgv1beta1.MetricsConfig{
				Spec: metricscfgv1beta1.MetricsConfigSpec{
					PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
						CollectPreviousData: &falseDef,
					},
				},
			}

			got, _ := getTimeRange(ctx, r, cr)
			Expect(got).To(Equal(original))
			Expect(got).ToNot(Equal(original.Add(-fourteenDayDuration)))
		})
	})

	Context("mocking QueryRange to test controller flow", func() {
		BeforeEach(func() {
			checkPVC = true

			mockCtrl = gomock.NewController(GinkgoT())
			mockpconn = mocks.NewMockPrometheusConnection(mockCtrl)

		})
		JustBeforeEach(func() {
			promConnTester = func(promcoll *collector.PrometheusCollector) error { return nil }
			promConnSetter = func(promcoll *collector.PrometheusCollector) error {
				promcoll.PromConn = mockpconn
				return nil
			}
		})
		It("failed to get prometheus config because of missing token", func() {
			resetReconciler(WithSecretOverride(false))

			t := time.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			timeRange := promv1.Range{
				Start: t,
				End:   t.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(model.Matrix{}, nil, nil).Times(0)

			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.Prometheus.ConfigError != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Prometheus.ConfigError).To(ContainSubstring("failed to get token"))
		})
		It("successfully queried but there was no data", func() {
			resetReconciler(WithSecretOverride(true))

			t := time.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			timeRange := promv1.Range{
				Start: t,
				End:   t.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(model.Matrix{}, nil, nil).MinTimes(1)

			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeFalse())
			Expect(fetched.Status.Reports.DataCollectionMessage).To(ContainSubstring("No data to report for the hour queried."))

		})
		It("2day retention period - successfully queried but there was no data on first day, but data on second", func() {
			resetReconciler(WithSecretOverride(true))

			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 2d"}
			createObject(ctx, testConfigMap)

			t := time.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			t2 := t.Truncate(24 * time.Hour).Add(-24 * time.Hour) // midnight yesterday
			timeRangeInitial := promv1.Range{
				Start: t2,
				End:   t2.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRangeInitial, gomock.Any()).Return(model.Matrix{}, nil, nil).MinTimes(1)
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(asModelMatrix(metricjson, nodeallocatablecpucores), nil, nil).MinTimes(1)

			instCopy.Spec.PrometheusConfig.CollectPreviousData = &trueDef
			instCopy.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization = &trueDef
			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeTrue())
		})
		It("2day retention period - end of 24 hr test", func() {
			resetReconciler(WithSecretOverride(true))
			now = func() time.Time { return time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour) }

			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 2d"}
			createObject(ctx, testConfigMap)

			t := now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			t2 := t.Truncate(24 * time.Hour).Add(-24 * time.Hour) // midnight yesterday
			timeRangeInitial := promv1.Range{
				Start: t2,
				End:   t2.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRangeInitial, gomock.Any()).Return(model.Matrix{}, nil, nil).MinTimes(1)
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(asModelMatrix(metricjson, nodeallocatablecpucores), nil, nil).MinTimes(1)

			instCopy.Spec.PrometheusConfig.CollectPreviousData = &trueDef
			instCopy.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization = &trueDef
			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.Prometheus.LastQuerySuccessTime.Equal(&metav1.Time{Time: t})
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeTrue())
			Expect(len(fetched.Status.Packaging.PackagedFiles)).To(Equal(1))
			Expect(fetched.Status.Packaging.ReportCount).ToNot(BeNil())
			Expect(*fetched.Status.Packaging.ReportCount).To(BeEquivalentTo(1))

			files, err := os.ReadDir(filepath.Join(".", "tmp", volumeMountName, "data"))
			Expect(err).To(BeNil())
			Expect(len(files)).To(Equal(0))
		})
		It("8day retention period - successfully queried but there was no data on first day, but data on all remaining days", func() {
			resetReconciler(WithSecretOverride(true))

			testConfigMap.Data = map[string]string{"config.yaml": "prometheusK8s:\n  retention: 8d"}
			createObject(ctx, testConfigMap)

			t := metav1.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			t2 := t.Truncate(24 * time.Hour).Add(-24 * 8 * time.Hour) // midnight 8 days ago
			timeRangeInitial := promv1.Range{
				Start: t2,
				End:   t2.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRangeInitial, gomock.Any()).Return(model.Matrix{}, nil, nil).MinTimes(1)
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(asModelMatrix(metricjson, nodeallocatablecpucores), nil, nil).MinTimes(1)

			instCopy.Spec.Packaging.MaxReports = 2
			instCopy.Spec.PrometheusConfig.CollectPreviousData = &trueDef
			instCopy.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization = &trueDef
			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.Prometheus.LastQuerySuccessTime.Equal(&metav1.Time{Time: t})
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeTrue())
			Expect(len(fetched.Status.Packaging.PackagedFiles)).To(Equal(2))
			Expect(fetched.Status.Packaging.ReportCount).ToNot(BeNil())
			Expect(*fetched.Status.Packaging.ReportCount).To(BeEquivalentTo(2))

		})
		It("query failed due to error", func() {
			resetReconciler(WithSecretOverride(true))

			t := time.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			timeRange := promv1.Range{
				Start: t,
				End:   t.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(model.Matrix{}, nil, errors.New("test error")).MinTimes(1)

			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeFalse())
			Expect(fetched.Status.Reports.DataCollectionMessage).To(ContainSubstring("test error"))

		})
		It("query returns node data only", func() {
			resetReconciler(WithSecretOverride(true))

			t := time.Now().UTC().Truncate(1 * time.Hour).Add(-1 * time.Hour)
			timeRange := promv1.Range{
				Start: t,
				End:   t.Add(59*time.Minute + 59*time.Second),
				Step:  time.Minute,
			}
			start := timeRange.Start.Add(1 * time.Second)
			end := start.Add(14*time.Minute + 59*time.Second)
			timeROS1 := end
			timeROS2 := timeROS1.Add(15 * time.Minute)
			timeROS3 := timeROS2.Add(15 * time.Minute)
			timeROS4 := timeROS3.Add(15 * time.Minute)
			// this mock is tightly coupled to the order in which the node queries are run and the first pod query
			gomock.InOrder(
				// node-allocatable-cpu-cores
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, nodeallocatablecpucores), nil, nil),
				// node-allocatable-memory-bytes
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, nodeallocatablememorybytes), nil, nil),
				// node-capacity-cpu-cores
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, nodecapacitycpucores), nil, nil),
				// node-capacity-memory-bytes
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, nodecapacitymemorybytes), nil, nil),
				// node-role
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, noderole), nil, nil),
				// node-labels
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, nodelabels), nil, nil),
				// pod-limit-cpu-cores
				mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(asModelMatrix(metricjson, podlimitcpucores), nil, nil),
			)
			// mock the rest of the Queries Anytimes
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), timeRange, gomock.Any()).Return(model.Matrix{}, nil, nil).AnyTimes()
			mockpconn.EXPECT().Query(gomock.Any(), gomock.Any(), timeROS1, gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()
			mockpconn.EXPECT().Query(gomock.Any(), gomock.Any(), timeROS2, gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()
			mockpconn.EXPECT().Query(gomock.Any(), gomock.Any(), timeROS3, gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()
			mockpconn.EXPECT().Query(gomock.Any(), gomock.Any(), timeROS4, gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()

			// Mock these again in case the CR is reconciled again
			mockpconn.EXPECT().QueryRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(model.Matrix{}, nil, nil).AnyTimes()
			mockpconn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(model.Vector{}, nil, nil).AnyTimes()

			instCopy.Spec.Upload.UploadToggle = &falseValue
			createObject(ctx, instCopy)

			fetched := &metricscfgv1beta1.MetricsConfig{}

			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: instCopy.Name, Namespace: namespace}, fetched)
				return fetched.Status.ClusterID != ""
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Reports.DataCollected).To(BeTrue())
		})
	})
})
