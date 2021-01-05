package archive

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kokumetricscfgv1alpha1 "github.com/project-koku/koku-metrics-operator/api/v1alpha1"
)

var (
	tenGi = *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI)
	// DefaultPVC is a basic PVC
	DefaultPVC = kokumetricscfgv1alpha1.EmbeddedPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		EmbeddedObjectMetadata: kokumetricscfgv1alpha1.EmbeddedObjectMetadata{
			Name: "koku-metrics-operator-data",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: tenGi,
				},
			},
		},
	}
)

// MakeVolumeClaimTemplate produces a template to create the PVC
func MakeVolumeClaimTemplate(e kokumetricscfgv1alpha1.EmbeddedPersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: e.APIVersion,
			Kind:       e.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              e.Name,
			Namespace:         "koku-metrics-operator",
			Labels:            e.Labels,
			Annotations:       e.Annotations,
			CreationTimestamp: metav1.Time{},
		},
		Spec:   e.Spec,
		Status: e.Status,
	}
	return &pvc
}
