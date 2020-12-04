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
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var namespace = "openshift-cost"
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

var _ = Describe("CostmanagementController", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		os.Mkdir("foo", 0777)
	})

	AfterEach(func() {

	})

	Describe("CostManagement CRD Handling", func() {
		Context("Process CRD resource", func() {
			It("should provide defaults for empty CRD case", func() {

				instance := costmgmtv1alpha1.CostManagement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      namePrefix + "empty",
						Namespace: namespace,
					},
					Spec: costmgmtv1alpha1.CostManagementSpec{
						Authentication: costmgmtv1alpha1.AuthenticationSpec{
							AuthType: costmgmtv1alpha1.Token,
						},
						Packaging: costmgmtv1alpha1.PackagingSpec{
							MaxSize: 100,
						},
						Upload: costmgmtv1alpha1.UploadSpec{
							UploadCycle:    &defaultUploadCycle,
							UploadToggle:   &defaultUploadToggle,
							IngressAPIPath: "/api/ingress/v1/upload",
							ValidateCert:   &defaultValidateCert,
						},
						Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
							CreateSource:   &defaultCreateSource,
							SourcesAPIPath: "/api/sources/v1.0/",
							CheckCycle:     &defaultCheckCycle,
						},
						PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
							SkipTLSVerification: &defaultSkipTLSVerify,
							SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
						},
						APIURL: "https://cloud.redhat.com",
					},
				}

				Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
				time.Sleep(time.Second * 5)

				fetched := &costmgmtv1alpha1.CostManagement{}

				// check the CRD was created ok
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
				Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
		})
		It("upload set to false case", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "uploadfalse",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType: costmgmtv1alpha1.Token,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &falseUpload,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Upload.UploadToggle).To(Equal(&falseUpload))
		})
		It("should find basic auth token for good basic auth CRD case", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "basic",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType:                 costmgmtv1alpha1.Basic,
						AuthenticationSecretName: authSecretName,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail for missing basic auth token for bad basic auth CRD case", func() {
			badAuth := "bad-auth"
			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "basicbad",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType:                 costmgmtv1alpha1.Basic,
						AuthenticationSecretName: badAuth,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuth))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should reflect source name in status for source info CRD case", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "sourceinfo",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType: costmgmtv1alpha1.Token,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						SourceName:     sourceName,
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			Expect(fetched.Status.Source.SourceName).To(Equal(sourceName))
		})
		It("should fail due to bad basic auth secret", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "badauth",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType:                 costmgmtv1alpha1.Basic,
						AuthenticationSecretName: badAuthSecretName,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail due to missing pass in auth secret", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "badpass",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType:                 costmgmtv1alpha1.Basic,
						AuthenticationSecretName: badAuthPassSecretName,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail due to missing auth secret name with basic set", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "missingname",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType: costmgmtv1alpha1.Basic,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			// Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(badAuthPassSecretName))
			Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should should fail token secret", func() {
			deletePullSecret(openShiftConfigNamespace, fakeDockerConfig())
			// createBadPullSecret(openShiftConfigNamespace)
			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "nopull",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType: costmgmtv1alpha1.Token,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			// Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should should fail token secret wrong data", func() {
			createBadPullSecret(openShiftConfigNamespace)
			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "nopulldata",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType: costmgmtv1alpha1.Token,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
			// Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
		It("should fail bc of missing cluster version", func() {
			deleteClusterVersion()

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "failurecv",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication: costmgmtv1alpha1.AuthenticationSpec{
						AuthType:                 costmgmtv1alpha1.Basic,
						AuthenticationSecretName: authSecretName,
					},
					Packaging: costmgmtv1alpha1.PackagingSpec{
						MaxSize: 100,
					},
					Upload: costmgmtv1alpha1.UploadSpec{
						UploadCycle:    &defaultUploadCycle,
						UploadToggle:   &defaultUploadToggle,
						IngressAPIPath: "/api/ingress/v1/upload",
						ValidateCert:   &defaultValidateCert,
					},
					Source: costmgmtv1alpha1.CloudDotRedHatSourceSpec{
						CreateSource:   &defaultCreateSource,
						SourcesAPIPath: "/api/sources/v1.0/",
						CheckCycle:     &defaultCheckCycle,
					},
					PrometheusConfig: costmgmtv1alpha1.PrometheusSpec{
						SkipTLSVerification: &defaultSkipTLSVerify,
						SvcAddress:          "https://thanos-querier.openshift-monitoring.svc:9091",
					},
					APIURL: "https://cloud.redhat.com",
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Expect(fetched.Status.Authentication.AuthType).To(Equal(costmgmtv1alpha1.Basic))
			// Expect(fetched.Status.Authentication.AuthenticationSecretName).To(Equal(authSecretName))
			// Expect(*fetched.Status.Authentication.AuthenticationCredentialsFound).To(BeTrue())
			// Expect(fetched.Status.APIURL).To(Equal(costmgmtv1alpha1.DefaultAPIURL))
			// Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})
	})
})
