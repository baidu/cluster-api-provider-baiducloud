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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"

	"github.com/baidu/baiducloud-sdk-go/bcc"
	"github.com/baidu/baiducloud-sdk-go/bce"
	"github.com/baidu/baiducloud-sdk-go/billing"
	"github.com/baidu/baiducloud-sdk-go/clientset"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ccecfgV1alpha1 "sigs.k8s.io/cluster-api-provider-baiducloud/pkg/apis/cceproviderconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/cert"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProviderName = "baidu"

	TagInstanceID        = "instanceID"
	TagInstanceStatus    = "instanceStatus"
	TagInstanceAdminPass = "instanceAdminPass"
	TagKubeletVersion    = "kubelet-version"

	TagMasterInstanceID = "masterInstanceID"
	TagMasterIP         = "masterIP"
)

var MachineActuator *CCEClient

type SSHCreds struct {
	user           string
	privateKeyPath string
}

type CCEClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type CCEClient struct {
	certificateAuthority *cert.CertificateAuthority
	computeService       CCEClientComputeService
	kubeadm              CCEClientKubeadm
	// TODO sa
	sshCreds      SSHCreds
	client        client.Client
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
}

type MachineActuatorParams struct {
	CertificateAuthority *cert.CertificateAuthority
	ComputeService       CCEClientComputeService
	Kubeadm              CCEClientKubeadm
	Client               client.Client
	// configgetter
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
}

// NewMachineActuator creates a new machine actuator
func NewMachineActuator(params MachineActuatorParams) (*CCEClient, error) {
	compuetService, err := getOrNewComputeServiceForMachine(params)
	if err != nil {
		glog.Errorf("create computeservice err, %+v", err)
		return nil, err
	}
	return &CCEClient{
		computeService: compuetService,
		client:         params.Client,
		eventRecorder:  params.EventRecorder,
		scheme:         params.Scheme,
	}, nil
}

// Create creates a new instance machine in the cluster
func (cce *CCEClient) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	glog.V(4).Infof("Create machine: %+v", machine.Name)
	instance, err := cce.instanceIfExists(cluster, machine)
	if err != nil {
		return err
	}

	if instance != nil {
		glog.Infof("Skipped creating a VM that already exists, instanceID %s", instance.InstanceID)
	}

	machineCfg, err := machineProviderFromProviderConfig(machine.Spec.ProviderSpec)
	if err != nil {
		glog.Errorf("parse machine config err: %s", err.Error())
		return err
	}
	glog.V(4).Infof("machine config: %+v", machineCfg)

	bccArgs := &bcc.CreateInstanceArgs{
		Name:    "cluster-api-" + machineCfg.Role,
		ImageID: machineCfg.ImageID, // ubuntu-16.04-amd64
		Billing: billing.Billing{
			PaymentTiming: "Postpaid",
		},
		CPUCount:           machineCfg.CPUCount,
		MemoryCapacityInGB: machineCfg.MemoryCapacityInGB,
		AdminPass:          machineCfg.AdminPass,
		PurchaseCount:      1,
		// InstanceType:       "10", // common 3 // TODO
	}

	// TODO support different regions
	instanceIDs, err := cce.computeService.Bcc().CreateInstances(bccArgs, nil)
	if err != nil {
		return err
	}

	if len(instanceIDs) != 1 {
		return fmt.Errorf("CreateVMError")
	}

	glog.Infof("Created a new VM, instanceID %s", instanceIDs[0])
	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = map[string]string{}
	}
	machine.ObjectMeta.Annotations[TagInstanceID] = instanceIDs[0]
	machine.ObjectMeta.Annotations[TagInstanceStatus] = "Created"
	machine.ObjectMeta.Annotations[TagInstanceAdminPass] = machineCfg.AdminPass
	machine.ObjectMeta.Annotations[TagKubeletVersion] = machine.Spec.Versions.Kubelet
	glog.V(4).Infof("new machine: %+v", machine.Name)
	cce.client.Update(context.Background(), machine)

	if machineCfg.Role == "master" {
		cluster.ObjectMeta.Annotations[TagMasterInstanceID] = instanceIDs[0]
		cce.client.Update(context.Background(), cluster)
	}

	// TODO deploy master
	// TODO deploy node

	time.Sleep(3 * time.Second)
	return nil
}

// Delete cleans a node
func (cce *CCEClient) Delete(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	glog.V(4).Infof("Delete machine: %+v", machine.Name)
	instance, err := cce.instanceIfExists(cluster, machine)
	if err != nil {
		return err
	}
	if instance == nil || len(instance.CreationTime) == 0 {
		glog.Infof("Skipped delete a VM that already does not exist")
		return nil
	}
	if err := cce.computeService.Bcc().DeleteInstance(instance.InstanceID, nil); err != nil {
		glog.Errorf("delete instance %s err: %+v", instance.InstanceID, err)
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}

// Exists checks the existances of some instance
func (cce *CCEClient) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	glog.V(4).Infof("Check machine: %+v", machine.Name)
	instance, err := cce.instanceIfExists(cluster, machine)
	if err != nil {
		return false, err
	}
	return (instance != nil), nil
}

// Update updates the some machine
func (cce *CCEClient) Update(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	// TODO
	return nil
}

// GetIP returns ip of some machine
func (cce *CCEClient) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	// TODO
	return "", nil
}

// GetKubeConfig returns config of some mahine
func (cce *CCEClient) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	// TODO
	return "", nil
}

func (cce *CCEClient) instanceIfExists(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (*bcc.Instance, error) {
	targetInstanceID := machine.GetAnnotations()[TagInstanceID]
	if len(targetInstanceID) == 0 {
		return nil, nil
	}
	glog.V(4).Infof("check existence of instance %s", targetInstanceID)
	instance, err := cce.computeService.Bcc().DescribeInstance(targetInstanceID, nil)
	if err != nil {
		glog.Errorf("DescribeInstance err: %+v", err.Error())
		berr, ok := err.(*bce.Error)
		if ok && berr.StatusCode == 404 {
			return &bcc.Instance{
				InstanceID: targetInstanceID,
			}, nil
		}
		return nil, err
	}
	return instance, nil
}

func getOrNewComputeServiceForMachine(params MachineActuatorParams) (CCEClientComputeService, error) {
	glog.V(4).Infof("create compute service")
	if params.ComputeService != nil {
		return params.ComputeService, nil
	}

	credential := &bce.Credentials{
		AccessKeyID:     os.Getenv("AccessKeyID"),
		SecretAccessKey: os.Getenv("SecretAccessKey"),
	}
	cfg := bce.NewConfig(credential)
	cfg.Region = "bj"
	clientSet, err := clientset.NewFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func machineProviderFromProviderConfig(providerConfig clusterv1.ProviderSpec) (*ccecfgV1alpha1.CCEMachineProviderConfig, error) {
	var config ccecfgV1alpha1.CCEMachineProviderConfig
	if err := yaml.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
