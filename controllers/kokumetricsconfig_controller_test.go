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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kokumetricscfgv1alpha1 "github.com/project-koku/koku-metrics-operator/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var namespace = "koku-metrics-operator"
var namePrefix = "cost-test-local-"
var clusterID = "10e206d7-a11a-403e-b835-6cff14e98b23"
var sourceName = "cluster-test"
var authSecretName = "cloud-dot-redhat"
var badAuthSecretName = "baduserpass"
var badAuthPassSecretName = "badpass"

// Defaults for spec
var defaultUploadCycle int64 = 360
var defaultCheckCycle int64 = 1440
var defaultUploadToggle bool = true
var falseUpload bool = false
var defaultCreateSource bool = false
var defaultSkipTLSVerify bool = true
var defaultValidateCert bool = true

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

func setup() error {
	type dirInfo struct {
		dirName  string
		files    []string
		dirMode  os.FileMode
		fileMode os.FileMode
	}
	testFiles := []string{"ocp_node_label.csv", "nonCSV.txt", "testFile.tar.gz"}
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
	fmt.Println("Setting up for packaging tests")
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
				_, err := Copy(directory.fileMode, filepath.Join("../packaging/test_files/", reportFile), filepath.Join(reportPath, reportFile))
				if err != nil {
					return fmt.Errorf("could not copy %s file: %v", reportFile, err)
				}
			}
		}
	}
	return nil
}

func shutdown() {
	fmt.Println("Tearing down for packaging tests")
	os.RemoveAll("/tmp/koku-metrics-operator-reports/")
}

var _ = Describe("KokuMetricsConfigController", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		shutdown()
	})

	AfterEach(func() {
		// shutdown()

	})

	Describe("KokuMetricsConfig CRD Handling", func() {
		Context("Process CRD resource", func() {
			It("should provide defaults for empty CRD case", func() {

				instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      namePrefix + "empty",
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
							UploadToggle:   &defaultUploadToggle,
							IngressAPIPath: "/api/ingress/v1/upload",
							ValidateCert:   &defaultValidateCert,
						},
						Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
							CreateSource:   &defaultCreateSource,
							SourcesAPIPath: "/api/sources/v1.0/",
							CheckCycle:     &defaultCheckCycle,
						},
						PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
							SkipTLSVerification: &defaultSkipTLSVerify,
							SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
						},
						APIURL: "https://cloud.redhat.com",
					},
				}

				Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
				time.Sleep(time.Second * 5)

				fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

				// check the CRD was created ok
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
		})
		It("upload set to false case", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "uploadfalse",
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
						UploadToggle:   &falseUpload,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadToggle).To(Equal(&falseUpload))
		})
		It("should find basic auth token for good basic auth CRD case", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "basic",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType:                 kokumetricscfgv1alpha1.Basic,
						AuthenticationSecretName: authSecretName,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail for missing basic auth token for bad basic auth CRD case", func() {
			badAuth := "bad-auth"
			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "basicbad",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType:                 kokumetricscfgv1alpha1.Basic,
						AuthenticationSecretName: badAuth,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuth))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should reflect source name in status for source info CRD case", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "sourceinfo",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType: kokumetricscfgv1alpha1.Token,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						SourceName:     sourceName,
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
		})
		It("should fail due to bad basic auth secret", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "badauth",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType:                 kokumetricscfgv1alpha1.Basic,
						AuthenticationSecretName: badAuthSecretName,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail due to missing pass in auth secret", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "badpass",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType:                 kokumetricscfgv1alpha1.Basic,
						AuthenticationSecretName: badAuthPassSecretName,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail due to missing auth secret name with basic set", func() {

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "missingname",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType: kokumetricscfgv1alpha1.Basic,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(kokumetricscfgv1alpha1.Basic))
			// Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(kokumetricscfgv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should should fail token secret", func() {
			deletePullSecret(openShiftConfigNamespace, fakeDockerConfig())
			// createBadPullSecret(openShiftConfigNamespace)
			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "nopull",
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
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should should fail token secret wrong data", func() {
			createBadPullSecret(openShiftConfigNamespace)
			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "nopulldata",
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
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail bc of missing cluster version", func() {
			deleteClusterVersion()

			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "failurecv",
					Namespace: namespace,
				},
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					Authentication: kokumetricscfgv1alpha1.AuthenticationSpec{
						AuthType:                 kokumetricscfgv1alpha1.Basic,
						AuthenticationSecretName: authSecretName,
					},
					Packaging: kokumetricscfgv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: kokumetricscfgv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.Basic))
			// Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should run through more of the flow", func() {
			err := setup()
			Expect(err, nil)
			createClusterVersion(clusterID)
			var uploadWait int64 = 0
			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "more",
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
						UploadToggle:   &defaultUploadToggle,
						UploadWait:     &uploadWait,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 10)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should check the status", func() {
			err := setup()
			Expect(err, nil)
			var uploadWait int64 = 0
			uploadTime := metav1.Now()
			instance := kokumetricscfgv1alpha1.KokuMetricsConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "status",
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
						UploadToggle:   &defaultUploadToggle,
						UploadWait:     &uploadWait,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
				Status: kokumetricscfgv1alpha1.KokuMetricsConfigStatus{
					Upload: kokumetricscfgv1alpha1.UploadStatus{
						LastSuccessfulUploadTime: uploadTime,
					},
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 20)

			fetched := &kokumetricscfgv1alpha1.KokuMetricsConfig{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		// It("should uploadagain", func() {
		// 	setup()
		// 	var uploadWait int64 = 0
		// 	uploadTime := metav1.Now()
		// 	instance :=  kokumetricscfgv1alpha1.KokuMetricsConfig{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      namePrefix + "uploadagain",
		// 			Namespace: namespace,
		// 		},
		// 		Spec:  kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
		// 			Authentication:  kokumetricscfgv1alpha1.AuthenticationSpec{
		// 				AuthType:  kokumetricscfgv1alpha1.Token,
		// 			},
		// 			Packaging:  kokumetricscfgv1alpha1.PackagingSpec{
		// 				MaxSize: 100,
		// 			},
		// 			Upload:  kokumetricscfgv1alpha1.UploadSpec{
		// 				UploadCycle:    &uploadWait,
		// 				UploadToggle:   &defaultUploadToggle,
		// 				UploadWait:     &uploadWait,
		// 				IngressAPIPath: "/api/ingress/v1/upload",
		// 				ValidateCert:   &defaultValidateCert,
		// 			},
		// 			Source:  kokumetricscfgv1alpha1.CloudDotRedHatSourceSpec{
		// 				CreateSource:   &defaultCreateSource,
		// 				SourcesAPIPath: "/api/sources/v1.0/",
		// 				CheckCycle:     &defaultCheckCycle,
		// 			},
		// 			PrometheusConfig:  kokumetricscfgv1alpha1.PrometheusSpec{
		// 				SkipTLSVerification: &defaultSkipTLSVerify,
		// 				SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
		// 			},
		// 			APIURL: "https://cloud.redhat.com",
		// 		},
		// 		Status:  kokumetricscfgv1alpha1.KokuMetricsConfigStatus{
		// 			Upload:  kokumetricscfgv1alpha1.UploadStatus{
		// 				LastSuccessfulUploadTime: uploadTime,
		// 			},
		// 		},
		// 	}

		// 	Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
		// 	time.Sleep(time.Second * 10)

		// 	fetched := & kokumetricscfgv1alpha1.KokuMetricsConfig{}

		// 	// check the CRD was created ok
		// 	Eventually(func() bool {
		// 		err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
		// 		return err == nil
		// 	}, timeout, interval).Should(BeTrue())

		// 	// Expect(fetched.Status.Authentication.AuthType).To(Equal( kokumetricscfgv1alpha1.DefaultAuthenticationType))
		// 	// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
		// 	// Expect(fetched.Status.APIURL).To(Equal( kokumetricscfgv1alpha1.DefaultAPIURL))
		// 	// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		// })
	})
})
