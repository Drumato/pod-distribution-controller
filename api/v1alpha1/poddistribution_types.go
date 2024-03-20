/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDistributionSpec defines the desired state of PodDistribution
type PodDistributionSpec struct {
	Selector PodDistributionSelector `json:"selector"`
	PDB      *PodDistributionPDBSpec `json:"minAvailable,omitEmpty"`
	// +optional
	AllowAugmentPodCollectionReplicas bool `json:"allowAugmentPodCollectionReplicas"`
}

type PodDistributionPDBSpec struct {
	// +optional
	MinAvailable *PodDistributionMinAvailableSpec `json:"minAvailable,omitEmpty"`
}

const (
	PodDistributionSelectorKindDeployment = "Deployment"
)

type PodDistributionSelector struct {
	// +kubebuilder:validation:Enum=Deployment
	Kind          string               `json:"kind"`
	LabelSelector metav1.LabelSelector `json:"labelSelector"`
}

type PodDistributionMinAvailableSpec struct {
	Auto string `json:"auto"`
	// +optional
	Policy           string `json:"policy"`
	AllowUndrainable bool   `json:"allowUndrainable"`
}

// PodDistributionStatus defines the observed state of PodDistribution
type PodDistributionStatus struct {
	TargetPodCollections []TargetPodCollection `json:"targetPodCollections"`
}

type TargetPodCollection struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PodDistribution is the Schema for the poddistributions API
type PodDistribution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodDistributionSpec   `json:"spec,omitempty"`
	Status PodDistributionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodDistributionList contains a list of PodDistribution
type PodDistributionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodDistribution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodDistribution{}, &PodDistributionList{})
}
