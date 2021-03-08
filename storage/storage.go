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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

var (
	tenGi = *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI)
	// DefaultPVC is a basic PVC
	DefaultPVC = kokumetricscfgv1beta1.EmbeddedPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		EmbeddedObjectMetadata: kokumetricscfgv1beta1.EmbeddedObjectMetadata{
			Name: "costmanagement-metrics-operator-data",
			Labels: map[string]string{
				"application": "costmanagement-metrics-operator",
			},
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

type volume struct {
	index  int
	volume *corev1.Volume
}

func (v *volume) isMounted() bool {
	return v.volume.PersistentVolumeClaim != nil
}

// Storage is a struct containing volume information
type Storage struct {
	Client    client.Client
	KMCfg     *kokumetricscfgv1beta1.CostManagementMetricsConfig
	Log       logr.Logger
	Namespace string
	PVC       *corev1.PersistentVolumeClaim

	vol *volume
}

func (s *Storage) getOrCreateVolume() error {
	ctx := context.Background()
	log := s.Log.WithValues("costmanagementmetricsconfig", "getOrCreateVolume")
	namespace := types.NamespacedName{
		Namespace: s.Namespace,
		Name:      s.PVC.Name}
	if err := s.Client.Get(ctx, namespace, s.PVC); err == nil {
		log.Info(fmt.Sprintf("PVC name %s already exists", s.PVC.Name))
		return nil
	}
	log.Info(fmt.Sprintf("attempting to create PVC name: %s", s.PVC.Name))
	return s.Client.Create(ctx, s.PVC)
}

func (s *Storage) getVolume(vols []corev1.Volume) error {
	for i, v := range vols {
		if v.Name == "costmanagement-metrics-operator-reports" {
			s.vol = &volume{index: i, volume: &v}
			if v.EmptyDir != nil {
				s.KMCfg.Status.Storage.VolumeType = v.EmptyDir.String()
			}
			if v.PersistentVolumeClaim != nil {
				s.KMCfg.Status.Storage.VolumeType = v.PersistentVolumeClaim.String()
			}
			return nil
		}
	}
	return fmt.Errorf("volume not found")
}

func (s *Storage) mountVolume(dep *appsv1.Deployment, depSpec *appsv1.DeploymentSpec, csv *operatorsv1alpha1.ClusterServiceVersion) (bool, error) {
	ctx := context.Background()
	s.vol.volume.EmptyDir = nil
	s.vol.volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: s.PVC.Name,
	}

	depSpec.Template.Spec.Volumes[s.vol.index] = *s.vol.volume
	var patch client.Patch
	var obj runtime.Object
	if csv != nil {
		obj = csv
		patch = client.MergeFrom(csv.DeepCopy())
		csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec = *depSpec
	} else {
		obj = dep
		patch = client.MergeFrom(dep.DeepCopy())
		dep.Spec = *depSpec
	}

	if err := s.Client.Patch(ctx, obj, patch); err != nil {
		return false, fmt.Errorf("failed to Patch %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, err)
	}
	return true, nil
}

// ConvertVolume converts the EmptyDir volume in deployment to PVC
func (s *Storage) ConvertVolume() (bool, error) {
	ctx := context.Background()
	log := s.Log.WithValues("costmanagementmetricsconfig", "ConvertVolume")

	log.Info("getting deployment")
	deployment := &appsv1.Deployment{}
	namespace := types.NamespacedName{
		Namespace: s.Namespace,
		Name:      "costmanagement-metrics-controller-manager"}
	if err := s.Client.Get(ctx, namespace, deployment); err != nil {
		return false, fmt.Errorf("unable to get Deployment: %v", err)
	}
	deployCp := deployment.DeepCopy()
	depSpec := deployCp.Spec.DeepCopy()

	var csv *operatorsv1alpha1.ClusterServiceVersion
	if len(deployCp.OwnerReferences) > 0 {
		owner := deployCp.OwnerReferences[0]
		log.Info(fmt.Sprintf("deployment is owned by: %s", owner.Name))
		csv = &operatorsv1alpha1.ClusterServiceVersion{}
		namespace := types.NamespacedName{
			Namespace: s.Namespace,
			Name:      owner.Name}
		if err := s.Client.Get(ctx, namespace, csv); err != nil {
			return false, fmt.Errorf("unable to get ClusterServiceVersion: %v", err)
		}
		depSpec = csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.DeepCopy()
	}

	log.Info("getting deployment volumes")
	if err := s.getVolume(deployCp.Spec.Template.Spec.Volumes); err != nil {
		return false, err
	}

	if s.vol.isMounted() && s.vol.volume.PersistentVolumeClaim.ClaimName == s.PVC.Name {
		log.Info(fmt.Sprintf("deployment volume is mounted to PVC name: %s", s.PVC.Name))
		s.KMCfg.Status.Storage.VolumeMounted = true
		return false, nil
	}

	log.Info("attempting to get or create PVC")
	if err := s.getOrCreateVolume(); err != nil {
		return false, fmt.Errorf("failed to get or create PVC: %v", err)
	}

	log.Info(fmt.Sprintf("attempting to mount deployment onto PVC name: %s", s.PVC.Name))
	return s.mountVolume(deployCp, depSpec, csv)
}

// MakeVolumeClaimTemplate produces a template to create the PVC
func MakeVolumeClaimTemplate(e kokumetricscfgv1beta1.EmbeddedPersistentVolumeClaim, namespace string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: e.APIVersion,
			Kind:       e.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Namespace:   namespace,
			Labels:      e.Labels,
			Annotations: e.Annotations,
		},
		Spec: e.Spec,
	}
}

// MakeEmbeddedPVC produces a template to create the PVC
func MakeEmbeddedPVC(pvc *corev1.PersistentVolumeClaim) *kokumetricscfgv1beta1.EmbeddedPersistentVolumeClaim {
	return &kokumetricscfgv1beta1.EmbeddedPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pvc.APIVersion,
			Kind:       pvc.Kind,
		},
		EmbeddedObjectMetadata: kokumetricscfgv1beta1.EmbeddedObjectMetadata{
			Name: pvc.Name,
		},
		Spec: pvc.Spec,
	}
}
