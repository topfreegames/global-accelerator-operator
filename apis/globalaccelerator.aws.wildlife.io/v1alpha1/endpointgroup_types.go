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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EndpointGroupSpec struct {
	DNSNames []string `json:"dnsNames,omitempty"`
}

type EndpointGroupStatus struct {
	GlobalAcceleratorARN string `json:"globalAcceleratorARN,omitempty"`
	ListenerARN          string `json:"listenerARN,omitempty"`
	EndpointGroupARN     string `json:"endpointGroupARN,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type EndpointGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EndpointGroupSpec   `json:"spec,omitempty"`
	Status EndpointGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type EndpointGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EndpointGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EndpointGroup{}, &EndpointGroupList{})
}
