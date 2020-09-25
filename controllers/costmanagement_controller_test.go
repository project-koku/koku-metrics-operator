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

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var namespace = "openshift-cost"
var namePrefix = "cost-test-local-"
var clusterID = "10e206d7-a11a-403e-b835-6cff14e98b23"
var authSecretName = "cloud-dot-redhat"

var _ = Describe("CostmanagementController", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
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
				}

				Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
				time.Sleep(time.Second * 10)

				fetched := &costmgmtv1alpha1.CostManagement{}

				// check the CRD was created ok
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(fetched.Status.Authentication).To(Equal(costmgmtv1alpha1.DefaultAuthenticationType))
				Expect(*fetched.Status.AuthenticationCredentialsFound).To(BeTrue())
				Expect(fetched.Status.IngressUrl).To(Equal(costmgmtv1alpha1.DefaultIngressUrl))
				Expect(fetched.Status.ClusterID).To(Equal(clusterID))
			})
		})
		It("should find basic auth token for good basic auth CRD case", func() {

			instance := costmgmtv1alpha1.CostManagement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namePrefix + "basic",
					Namespace: namespace,
				},
				Spec: costmgmtv1alpha1.CostManagementSpec{
					Authentication:           costmgmtv1alpha1.Basic,
					AuthenticationSecretName: authSecretName,
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 10)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.AuthenticationSecretName).To(Equal(authSecretName))
			Expect(*fetched.Status.AuthenticationCredentialsFound).To(BeTrue())
			Expect(fetched.Status.IngressUrl).To(Equal(costmgmtv1alpha1.DefaultIngressUrl))
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
					Authentication:           costmgmtv1alpha1.Basic,
					AuthenticationSecretName: badAuth,
				},
			}

			Expect(k8sClient.Create(ctx, &instance)).Should(Succeed())
			time.Sleep(time.Second * 10)

			fetched := &costmgmtv1alpha1.CostManagement{}

			// check the CRD was created ok
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Status.Authentication).To(Equal(costmgmtv1alpha1.Basic))
			Expect(fetched.Status.AuthenticationSecretName).To(Equal(badAuth))
			Expect(*fetched.Status.AuthenticationCredentialsFound).To(BeFalse())
			Expect(fetched.Status.IngressUrl).To(Equal(costmgmtv1alpha1.DefaultIngressUrl))
			Expect(fetched.Status.ClusterID).To(Equal(clusterID))
		})

	})
})
