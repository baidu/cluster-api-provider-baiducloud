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

// The MachineRole indicates the purpose of the Machine, and will determine
// what software and configuration will be used when provisioning and managing
// the Machine. A single Machine may have more than one role, and the list and
// definitions of supported roles is expected to evolve over time.
//
// Currently, only two roles are supported: Master and Node. In the future, we
// expect user needs to drive the evolution and granularity of these roles,
// with new additions accommodating common cluster patterns, like dedicated
// etcd Machines.
//
//                 +-----------------------+------------------------+
//                 | Master present        | Master absent          |
// +---------------+-----------------------+------------------------|
// | Node present: | Install control plane | Join the cluster as    |
// |               | and be schedulable    | just a node            |
// |---------------+-----------------------+------------------------|
// | Node absent:  | Install control plane | Invalid configuration  |
// |               | and be unschedulable  |                        |
// +---------------+-----------------------+------------------------+

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CCEMachineProviderConfig is the Schema for the CCEMachineProviderConfigs API
// +k8s:openapi-gen=true
type CCEMachineProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

	Role        string `json:"role"` // master or node
	ClusterID   string `json:"clusterId"`
	ClusterName string `json:"clusterName"`

	ImageID               string `json:"imageId"`
	CPUCount              int    `json:"cpuCount"`
	MemoryCapacityInGB    int    `json:"memoryCapacityInGB"`
	RootDiskSizeInGB      int    `json:"rootDiskSizeInGb,omitempty"`
	RootDiskStorageType   int    `json:"rootDiskStorageType,omitempty"`
	NetworkCapacityInMbps int    `json:"networkCapacityInMbps,omitempty"`
	Name                  string `json:"name,omitempty"`
	AdminPass             string `json:"adminPass,omitempty"`
	ZoneName              string `json:"zoneName,omitempty"`
	SubnetID              string `json:"subnetId,omitempty"`
	SecurityGroupID       string `json:"securityGroupId,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CCEMachineProviderConfigList contains a list of CCEMachineProviderConfig
type CCEMachineProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CCEMachineProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CCEMachineProviderConfig{}, &CCEMachineProviderConfigList{})
}
