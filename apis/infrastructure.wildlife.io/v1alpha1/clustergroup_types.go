/*
Copyright 2022.

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

type ClusterGroupSpec struct {
	Clusters []Cluster `json:"clusters,omitempty"`
}

type Cluster struct {
	Name           string                 `json:"name,omitempty"`
	CredentialsRef corev1.ObjectReference `json:"credentialsRef,omitempty"`
}

type ClusterGroupStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type ClusterGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterGroupSpec   `json:"spec,omitempty"`
	Status ClusterGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type ClusterGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterGroup{}, &ClusterGroupList{})
}
