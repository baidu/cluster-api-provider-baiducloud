# Cluster Api Provider Baiducloud

[![Build Status](https://travis-ci.org/baidu/cluster-api-provider-baiducloud.svg?branch=master)](https://travis-ci.org/baidu/cluster-api-provider-baiducloud)
[![Go Report Card](https://goreportcard.com/badge/github.com/baidu/cluster-api-provider-baiducloud)](https://goreportcard.com/report/github.com/baidu/cluster-api-provider-baiducloud)

## Components

### Binaries

* cluster-api-controller: The one which has no dependencies with baiducloud provider, which includes the cluster controller and the machine controller.

### Dockerization and helm/charts for deployment

We can dockerize the binaries and use helm/charts to deploy these three controllers into the cluster later.


## Installation

### Prerequisites

#### Local Binaries
`kubectl`
`sshpass`


### Build & Run manager

We just assume that you currently have a running Kubernetes cluster locally or in some public cloud, and you have your kubeconfig file under ~/.kube/config

```bash
~ glide install --strip-vendor
~ go build -o manager sigs.k8s.io/cluster-api-provider-baiducloud/cmd/manager
~ export SecretAccessKey=YOUR_ACCESS_KEY && export AccessKeyID=YOUR_KEY_ID
~ kubectl apply -f config/crds/cluster.yaml
~ kubectl apply -f config/crds/machine.yaml
~ kubectl apply -f config/crds/machineset.yaml
~ kubectl apply -f config/crds/machinedeployment.yaml
~ ./manager -kubeconfig ~/.kube/config -alsologtostderr -v 4
```

### Run an example

```bash
~ kubectl apply -f config/samples/cluster.yaml
~ kubectl apply -f config/samples/machinedeployments.yaml
```

## Usage

### Cluster Resource Creation Example

```bash

apiVersion: "cluster.k8s.io/v1alpha1"
kind: Cluster
metadata:
  name: cluster-sample
spec:
    clusterNetwork:
        services:
            cidrBlocks: ["10.96.0.0/16"]
        pods:
            cidrBlocks: ["100.10.0.0/16"]
        serviceDomain: "cluster.local"
    providerSpec:
      value:
        apiVersion: "cceproviderconfig/v1alpha1"
        kind: "CCEClusterProviderConfig"
        clusterName: "cluster-test3"
        clusterCIDR: "172.30.0.0/19"
```

### MachineDeployment Resource Creation Example

```bash
apiVersion: "cluster.k8s.io/v1alpha1"
kind: MachineDeployment
metadata:
  name: sample-master
spec:
  replicas: 1
  selector:
    matchLabels:
      foo: bar
  template:
    metadata:
      labels:
        foo: bar
    spec:
      providerSpec:
        value:
          apiVersion: "cceproviderconfig/v1alpha1"
          kind: "CCEMachineProviderConfig"
          clusterName: "cluster-test3"
          role: "master"
          adminPass: "testpw123!"
          imageId: "m-8WV4kRlN" # ubuntu 16.04 lts amd64
          cpuCount: 2
          memoryCapacityInGB: 2
      versions:
        kubelet: 1.12.3
        controlPlane: 1.12.3
  strategy:
    type: "RollingUpdate"
    rollingUpdate:
      maxUnavailable: "30%"
      maxSurge: "30%"
  minReadySeconds: 2
---
apiVersion: "cluster.k8s.io/v1alpha1"
kind: MachineDeployment
metadata:
  name: sample-nodes
spec:
  replicas: 3
  selector:
    matchLabels:
      foo: bar
  template:
    metadata:
      labels:
        foo: bar
    spec:
      providerSpec:
        value:
          apiVersion: "cceproviderconfig/v1alpha1"
          kind: "CCEMachineProviderConfig"
          clusterName: "cluster-test3"
          role: "node"
          adminPass: "testpw123!"
          imageId: "m-8WV4kRlN"
          cpuCount: 2
          memoryCapacityInGB: 2
      versions:
        kubelet: 1.12.3
  strategy:
    type: "RollingUpdate"
    rollingUpdate:
      maxUnavailable: "30%"
      maxSurge: "30%"
  minReadySeconds: 2

```
