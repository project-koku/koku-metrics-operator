//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package storage

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var (
	namespace = fmt.Sprintf("%s-metrics-operator", metricscfgv1beta1.NamePrefix)
	emptyDir  = &corev1.Volume{
		Name:         volumeMountName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	emptyDirWrong = &corev1.Volume{
		Name:         "wrong-mount",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	persistVC = &corev1.Volume{
		Name: volumeMountName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: volumeClaimName}}}
	persistVCfake = &corev1.Volume{
		Name: volumeMountName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "not-the-right-one"}}}
	volMount = &corev1.VolumeMount{
		Name:      volumeMountName,
		MountPath: dirconfig.MountPath,
	}
	volMountWrong = &corev1.VolumeMount{
		Name:      "wrong-mount",
		MountPath: dirconfig.MountPath,
	}
	// For the following deployments, the only things of importance are the ObjectMeta, Volumes, VolumeMounts.
	// All the other definitions are boilerplate so that the deployment will be created successfully.
	deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
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
			Name:      deploymentName,
			Namespace: namespace,
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
	csv = &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-csv",
			Namespace: namespace,
		},
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			DisplayName: "test-csv",
			InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
				StrategyName: "test-strategy",
				StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
					DeploymentSpecs: []operatorsv1alpha1.StrategyDeploymentSpec{
						{
							Name: "test-deployment",
							Spec: deployment.Spec,
						},
					},
				},
			},
		},
	}
	owner = metav1.OwnerReference{
		APIVersion: "operators.coreos.com/v1alpha1",
		Kind:       "ClusterServiceVersion",
		Name:       "test-csv",
		UID:        "1c5738c0-2691-4e60-b2fd-6e056327ac89",
	}
)

func int32Ptr(i int32) *int32 { return &i }

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := m.Run()
	os.Exit(code)
}

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

func TestMakeEmbeddedPVC(t *testing.T) {
	t.Run("embedded PVC does not contain annotations", func(t *testing.T) {
		pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
		got := MakeEmbeddedPVC(pvc)
		if got.Labels != nil {
			t.Errorf("embedded PVC should not have labels. got: %v", got.Labels)
		}
		if pvc.Name != got.Name {
			t.Errorf("unexpected PVC name. got: %s, want %s", got.Name, pvc.Name)
		}
	})
}

var _ = Describe("Storage Tests", func() {

	BeforeEach(func() {
		// failed test runs that do not clean up leave resources behind.
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.PersistentVolumeClaim{}, client.InNamespace(namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace(namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &operatorsv1alpha1.ClusterServiceVersion{}, client.InNamespace(namespace))).Should(Succeed())
	})

	AfterEach(func() {

	})
	Context("Deployment owned by CSV", func() {
		Describe("deployment does exist", func() {
			It("can find the deployment but CSV is missing", func() {
				// csvCp := csv.DeepCopy()
				// Expect(k8sClient.Create(ctx, csvCp)).Should(Succeed())

				depCp := deployment.DeepCopy()
				depCp.OwnerReferences = []metav1.OwnerReference{owner}
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).ToNot(BeNil())
				Expect(mountEst).To(BeFalse())
			})
			It("can find the deployment and CSV exists", func() {
				csvCp := csv.DeepCopy()
				Expect(k8sClient.Create(ctx, csvCp)).Should(Succeed())

				depCp := deployment.DeepCopy()
				depCp.OwnerReferences = []metav1.OwnerReference{owner}
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).To(BeNil())
				Expect(mountEst).To(BeTrue())
			})
		})
	})

	Context("Deployment not owned by CSV", func() {
		Describe("deployment does not exist", func() {
			It("cannot find the deployment", func() {
				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client: k8sClient,
					CR:     cr,
					PVC:    pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).ToNot(BeNil())
				Expect(mountEst).To(BeFalse())
			})
		})
		Describe("deployment does exist", func() {
			It("successfully establishes the mount", func() {
				depCp := deployment.DeepCopy()
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())
				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).To(BeNil())
				Expect(mountEst).To(BeTrue())
			})
			It("connot find the volume", func() {
				// replace the correct vol mount with an incorrect one
				depCp := deployment.DeepCopy()
				depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*emptyDirWrong}
				depCp.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{*volMountWrong}
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).ToNot(BeNil())
				Expect(mountEst).To(BeFalse())
			})
			It("deployment has no volumes at all", func() {
				// replace the correct vol mount with an incorrect one
				depCp := deploymentNoVolume.DeepCopy()
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).ToNot(BeNil())
				Expect(mountEst).To(BeFalse())
			})
			It("volume is already mounted", func() {
				depCp := deployment.DeepCopy()
				depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*persistVC}
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).To(BeNil())
				Expect(mountEst).To(BeFalse())
			})
			It("volume is already mounted but does not match spec", func() {
				pvcCp := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				pvcCp.Name = "not-the-right-one"
				Expect(k8sClient.Create(ctx, pvcCp)).Should(Succeed())
				depCp := deployment.DeepCopy()
				depCp.Spec.Template.Spec.Volumes = []corev1.Volume{*persistVCfake}
				Expect(k8sClient.Create(ctx, depCp)).Should(Succeed())

				cr := &metricscfgv1beta1.MetricsConfig{}
				pvc := MakeVolumeClaimTemplate(DefaultPVC, namespace)
				s := &Storage{
					Client:    k8sClient,
					CR:        cr,
					Namespace: namespace,
					PVC:       pvc,
				}

				mountEst, err := s.ConvertVolume()
				Expect(err).To(BeNil())
				Expect(mountEst).To(BeTrue())
			})
		})
	})
})
