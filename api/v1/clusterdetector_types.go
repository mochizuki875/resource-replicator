/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterDetectorSpec defines the desired state of ClusterDetector
type ClusterDetectorSpec struct {
	Context string `json:"context,omitempty"`
	Cluster string `json:"cluster,omitempty"`
	User    string `json:"user,omitempty"`
}

// ClusterDetectorStatus defines the observed state of ClusterDetector
type ClusterDetectorStatus struct {
	ClusterStatus string `json:"clusterstatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="CONTEXT",type="string",JSONPath=".spec.context"
//+kubebuilder:printcolumn:name="CLUSTER",type="string",JSONPath=".spec.cluster"
//+kubebuilder:printcolumn:name="USER",type="string",JSONPath=".spec.user"
//+kubebuilder:printcolumn:name="CLUSTERSTATUS",type="string",JSONPath=".status.clusterstatus"

// ClusterDetector is the Schema for the clusterdetectors API
type ClusterDetector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDetectorSpec   `json:"spec,omitempty"`
	Status ClusterDetectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterDetectorList contains a list of ClusterDetector
type ClusterDetectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDetector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterDetector{}, &ClusterDetectorList{})
}
