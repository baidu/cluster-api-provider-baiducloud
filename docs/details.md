# Details for cluster api provider baiducloud

## Components

### Binaries

* cluster-api-controller: The one which has no dependencies with baiducloud provider.
* cluster-controller: baiducloud cluster controller
* machine-controller: baiducloud machine controller

Those three controllers will be using Kubernetes native apis, they will be wrapped as CRD. 

### Dockerization and helm/charts for deployment

We can dockerize the binaries and use helm/charts to deploy these three controllers into the cluster.

## Installation

### CRDs

```bash

# kubectl get CustomResourceDefinition
NAME                                    AGE
clusters.cluster.k8s.io                 1d
machines.cluster.k8s.io                 1d
machinesets.cluster.k8s.io              1d
machinedeployments.cluster.k8s.io       1d
```

### Secrets

```bash

# cat config/yaml/baiducloud-api-secret.yaml 
apiVersion: v1
kind: Secret
metadata:
  name: baiducloud-api-secret
type: Opaque
data:
  SecretId: 'xxx'
  SecretKey: 'xxx'
```

### Controllers

```bash

# kubectl get deployment
NAME                            DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
clusterapi-generic-controller   1         1         1            1           1d
cce-cluster-controller          1         1         1            1           1d
cce-machine-controller          1         1         1            1           1d
```

## Usage

### Cluster Resource Creation Example

```bash

apiVersion: "cluster.k8s.io/v1alpha1"
kind: Cluster
metadata:
  name: $CLUSTER_NAME
  namespace: $NAMESPACE
spec:
    clusterNetwork:
        services:
            cidrBlocks: ["10.96.0.0/12"]
        pods:
            cidrBlocks: ["10.244.0.0/16"]
        serviceDomain: "cluster.local"
    providerConfig:
      value:
        apiVersion: "baiducloudproviderconfig/v1alpha1"
        kind: "BaiduCloudClusterProviderConfig"
```

### Machine Resource Creation Example

```bash

- apiVersion: "cluster.k8s.io/v1alpha1"
  kind: Machine
  metadata:
    generateName: $MASTER_NAME
    namespace: $NAMESPACE
    labels:
      set: master
  spec:
    providerConfig:
      value:
        region: "$REGION"
        size: "s-2vcpu-2gb"
        image: "ubuntu-18-04-x64"
        tags:
        - "machine-1"
        sshPublicKeys:
        - "ssh-rsa AAAA"
        private_networking: true
        backups: false
        ipv6: false
        # must be disabled for coreos instances.
        monitoring: true
    versions:
      controlPlane: 1.11.3
      kubelet: 1.11.3
```

### Demo

#### Build & Run manager

```bash
~ glide install --trip-vendor
~ go build -o manager sigs.k8s.io/cluster-api-provider-baiducloud/cmd/manager
~ export SecretAccessKey=YOUR_ACCESS_KEY && export AccessKeyID=YOUR_KEY_ID
~ kubectl apply -f config/crds/cluster.yaml
~ kubectl apply -f config/crds/machine.yaml
~ ./manager -kubeconfig ~/.kube/config -alsologtostderr -v 4
```

#### Run an example

```bash
~ kubectl apply -f config/samples/cluster.yaml
~ kubectl apply -f config/samples/machine.yaml
```