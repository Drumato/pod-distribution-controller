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

// LabelerSpec defines the desired state of Labeler
type LabelerSpec struct {
	LabelRules []LabelRule `json:"labelRule"`
}

type LabelRule struct {
	// +kubebuilder:validation:Enum=Deployment
	Kind          string `json:"kind"`
	CELExpression string `json:"celExpression"`
}

// LabelerStatus defines the observed state of Labeler
type LabelerStatus struct {
	TargetPodCollections []TargetPodCollection `json:"targetPodCollections"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Labeler is the Schema for the labelers API
type Labeler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LabelerSpec   `json:"spec,omitempty"`
	Status LabelerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LabelerList contains a list of Labeler
type LabelerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Labeler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Labeler{}, &LabelerList{})
}
