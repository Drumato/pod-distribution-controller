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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDistributionSpec defines the desired state of PodDistribution
type PodDistributionSpec struct {
	// +optional
	Distribution *DistributionSpec       `json:"distribution"`
	Selector     PodDistributionSelector `json:"selector"`
	// +optional
	PDB *PodDistributionPDBSpec `json:"pdb,omitEmpty"`
	// +optional
	AllowAugmentPodCollectionReplicas bool `json:"allowAugmentPodCollectionReplicas"`
}

type DistributionSpec struct {
	// +optional
	Pod DistributionPodSpec `json:"pod"`
	// +optional
	Node DistributionNodeSpec `json:"node"`
}

type DistributionPodSpec struct {
	// +optional
	TopologySpreadConstaints []DistributionPodTopologySpreadConstraintSpec `json:"topologySpreadConstaints"`
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
	// +optional
	Affinity *corev1.PodAffinity `json:"affinity,omitempty"`
	// +optional
	AntiAffinity *corev1.PodAntiAffinity `json:"antiAffinity,omitempty"`
}

type PodTopologySpreadConstraintSpecAutoMode string

const (
	PodTopologySpreadConstraintSpecAutoRegion = "region"
	PodTopologySpreadConstraintSpecAutoZone   = "zone"
	PodTopologySpreadConstraintSpecAutoNode   = "node"
)

type DistributionPodTopologySpreadConstraintSpec struct {
	// +optional
	Auto *DistributionPodTopologySpreadConstraintAutoSpec `json:"auto,omitempty"`
	// +optional
	Manual corev1.TopologySpreadConstraint `json:"manual"`
}

type DistributionPodTopologySpreadConstraintAutoSpec struct {
	// +kubebuilder:validation:Enum=region/zone/node
	Mode              string                               `json:"mode"`
	WhenUnsatisfiable corev1.UnsatisfiableConstraintAction `json:"whenUnsatisfiable"`
}

type DistributionNodeSpec struct {
	Name     string               `json:"name,omitempty"`
	Selector map[string]string    `json:"selector,omitempty"`
	Affinity *corev1.NodeAffinity `json:"affinity,omitempty"`
}

type PodDistributionPDBSpec struct {
	// +optional
	MinAvailable *PodDistributionMinAvailableSpec `json:"minAvailable,omitEmpty"`
	// +optional
	MaxUnavailable *PodDistributionMaxUnavailableSpec `json:"maxUnavailable,omitEmpty"`
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
	// +optional
	Auto string `json:"auto"`
	// +optional
	Policy string `json:"policy"`
	// +optional
	AllowUndrainable bool `json:"allowUndrainable"`
}

type PodDistributionMaxUnavailableSpec struct {
	// +optional
	Policy string `json:"policy"`
	// +optional
	AllowUnavailable bool `json:"allowUnavailable"`
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
