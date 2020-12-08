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

// Defaults for spec
var defaultUploadCycle int64 = 360
var defaultCheckCycle int64 = 1440
var defaultUploadToggle bool = true
var defaultCreateSource bool = false
var defaultSkipTLSVerify bool = true
var defaultValidateCert bool = true

var _ = Describe("KokuMetricsConfigController", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that do not clean up leave resources behind.
	})

	AfterEach(func() {

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
	})
})
