# CRD Examples

## Cluster

```yaml

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: clusters.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: Cluster
    plural: clusters
  scope: Namespaced
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
```

## Machine

```yaml

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: machines.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: Machine
    plural: machines
  scope: Namespaced
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []

```

## MachineDeployment

## MachineSet