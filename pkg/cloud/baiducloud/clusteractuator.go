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

package baiducloud

import (
	"github.com/baidu/baiducloud-sdk-go/bce"
	"github.com/baidu/baiducloud-sdk-go/clientset"
	"github.com/golang/glog"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type CCEClusterClient struct {
	computeService CCEClientComputeService
	client         client.Client
}

type ClusterActuatorParams struct {
	ComputeService CCEClientComputeService
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) (*CCEClusterClient, error) {
	computeService, err := getOrNewComputeServiceForCluster(params)
	if err != nil {
		return nil, err
	}
	return &CCEClusterClient{
		computeService: computeService,
		client:         m.GetClient(),
	}, nil
}

func (cce *CCEClusterClient) Reconcile(cluster *clusterv1.Cluster) error {
	glog.Infof("Reconciling cluster %v.", cluster.Name)
	return nil
}

func (cce *CCEClusterClient) Delete(cluster *clusterv1.Cluster) error {
	glog.Infof("Deleting cluster %v", cluster.Name)
	return nil
}

func getOrNewComputeServiceForCluster(params ClusterActuatorParams) (CCEClientComputeService, error) {
	if params.ComputeService != nil {
		return params.ComputeService, nil
	}

	cfg := bce.NewConfigWithParams("ak", "sk", "region")
	clientSet, err := clientset.NewFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}
