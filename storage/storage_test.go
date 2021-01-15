/*


Copyright 2021 Red Hat, Inc.

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

package storage

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kokumetricscfgv1alpha2 "github.com/project-koku/koku-metrics-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kokuMetricsCfgNamespace = "koku-metrics-operator"
	emptyDir                = &corev1.Volume{
		Name:         "koku-metrics-operator-reports",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	emptyDirWrong = &corev1.Volume{
		Name:         "wrong-mount",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	persistVC = &corev1.Volume{
		Name: "koku-metrics-operator-reports",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "koku-metrics-operator-data"}}}
	persistVCfake = &corev1.Volume{
		Name: "koku-metrics-operator-reports",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "not-the-right-one"}}}
	volMount = &corev1.VolumeMount{
		Name:      "koku-metrics-operator-reports",
		MountPath: "/tmp/koku-metrics-operator-reports",
	}
	volMountWrong = &corev1.VolumeMount{
		Name:      "wrong-mount",
		MountPath: "/tmp/koku-metrics-operator-reports",
	}
	// For the following deployments, the only things of importance are the ObjectMeta, Volumes, VolumeMounts.
	// All the other definitions are boilerplate so that the deployment will be created successfully.
	deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "koku-metrics-controller-manager",
			Namespace: kokuMetricsCfgNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         "web",
							Image:        "nginx:1.12",
							VolumeMounts: []corev1.VolumeMount{*volMount},
						},
					},
					Volumes: []corev1.Volume{*emptyDir},
				},
			},
		},
	}
	deploymentNoVolume = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "koku-metrics-controller-manager",
			Namespace: kokuMetricsCfgNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
						},
					},
				},
			},
		},
	}
)

func int32Ptr(i int32) *int32 { return &i }

func TestIsMounted(t *testing.T) {
	isMountedTests := []struct {
		name string
		vol  volume
		want bool
	}{
		{
			name: "volume is not a PVC - is not mounted",
			vol:  volume{volume: emptyDir},
			want: false,
		},
		{
			name: "volume is a PVC - is mounted",
			vol:  volume{volume: persistVC},
			want: true,
		},
	}
	for _, tt := range isMountedTests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vol.isMounted()
			if got != tt.want {
				t.Errorf("%s got %t want %t", tt.name, got, tt.want)
			}
		})
	}
}

var _ = Describe("Storage Tests", func() {

	BeforeEach(func() {
		// failed test runs that do not clean up leave resources behind.
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.PersistentVolumeClaim{}, client.InNamespace(kokuMetricsCfgNamespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace(kokuMetricsCfgNamespace))).Should(Succeed())
	})

	AfterEach(func() {

	})

	Describe("deployment does not exist", func() {
		It("cannot find the deployment", func() {
			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).ToNot(BeNil())
			Expect(mountEst).To(BeFalse())
		})
	})
	Describe("deployment does exist", func() {
		It("successfully establishes the mount", func() {
			depCp := deployment.DeepCopy()
			Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())
			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).To(BeNil())
			Expect(mountEst).To(BeTrue())
		})
		It("connot find the volume", func() {
			// replace the correct vol mount with an incorrect one
			depCp := deployment.DeepCopy()
			depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*emptyDirWrong}
			depCp.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{*volMountWrong}
			Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).ToNot(BeNil())
			Expect(mountEst).To(BeFalse())
		})
		It("deployment has no volumes at all", func() {
			// replace the correct vol mount with an incorrect one
			depCp := deploymentNoVolume.DeepCopy()
			Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).ToNot(BeNil())
			Expect(mountEst).To(BeFalse())
		})
		It("volume is already mounted", func() {
			depCp := deployment.DeepCopy()
			depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*persistVC}
			Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).To(BeNil())
			Expect(mountEst).To(BeFalse())
		})
		It("volume is already mounted but does not match spec", func() {
			pvcCp := MakeVolumeClaimTemplate(DefaultPVC)
			pvcCp.Name = "not-the-right-one"
			Expect(k8sClient.Create(ctx, pvcCp)).Should(Succeed())
			depCp := deployment.DeepCopy()
			depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*persistVCfake}
			Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

			kmCfg := &kokumetricscfgv1alpha2.KokuMetricsConfig{}
			pvc := MakeVolumeClaimTemplate(DefaultPVC)
			s := &Storage{
				Client: k8sClient,
				Log:    testLogger,
				PVC:    pvc,
			}

			mountEst, err := s.ConvertVolume(kmCfg)
			Expect(err).To(BeNil())
			Expect(mountEst).To(BeTrue())
		})
	})
})
