/*
Copyright 2018 The Kubernetes Authors.

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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CCEClusterProviderConfig is the Schema for the cceclusterproviderconfigs API
// +k8s:openapi-gen=true
type CCEClusterProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ClusterName    string `json:"clusterName"`
	ClusterCIDR    string `json:"clusterCIDR"`
	ClusterVersion string `json:"clusterVersion"`
	VpcID          string `json:"vpcId"`
	Region         string `json:"region"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CCEClusterProviderConfigList contains a list of CCEClusterProviderConfig
type CCEClusterProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CCEClusterProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CCEClusterProviderConfig{}, &CCEClusterProviderConfigList{})
}
