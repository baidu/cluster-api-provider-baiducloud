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
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"

	"github.com/baidu/baiducloud-sdk-go/bcc"
	"github.com/baidu/baiducloud-sdk-go/bce"
	"github.com/baidu/baiducloud-sdk-go/billing"
	"github.com/baidu/baiducloud-sdk-go/clientset"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	ccecfgV1alpha1 "sigs.k8s.io/cluster-api-provider-baiducloud/pkg/apis/cceproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/cloud/utils"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/cert"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProviderName = "baidu"

	TagInstanceRole      = "instanceRole"
	TagInstanceID        = "instanceID"
	TagInstanceStatus    = "instanceStatus"
	TagInstanceAdminPass = "instanceAdminPass"
	TagKubeletVersion    = "kubelet-version"

	TagClusterToken     = "clusterToken"
	TagMasterInstanceID = "masterInstanceID"
	TagMasterIP         = "masterIP"
)

// MachineActuator is the client of cloud provider baidu
var MachineActuator *CCEClient

// SSHCreds ssh credentials
// TODO
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
		kubeadm:        getOrNewKubeadm(params),
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
		Name:    machine.Name,
		ImageID: machineCfg.ImageID, // ubuntu-16.04-amd64
		Billing: billing.Billing{
			PaymentTiming: "Postpaid",
		},
		CPUCount:              machineCfg.CPUCount,
		MemoryCapacityInGB:    machineCfg.MemoryCapacityInGB,
		AdminPass:             machineCfg.AdminPass,
		PurchaseCount:         1,
		InstanceType:          "N3", // Normal 3
		NetworkCapacityInMbps: 1,    //EIP bandwidth
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
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = map[string]string{}
	}
	machine.ObjectMeta.Annotations[TagInstanceID] = instanceIDs[0]
	machine.ObjectMeta.Annotations[TagInstanceStatus] = "Created"
	machine.ObjectMeta.Annotations[TagInstanceAdminPass] = machineCfg.AdminPass
	machine.ObjectMeta.Annotations[TagKubeletVersion] = machine.Spec.Versions.Kubelet

	token, err := cce.getKubeadmToken()
	if err != nil {
		glog.Errorf("getKubeadmToken err: %+v", err)
		return err
	}

	if machineCfg.Role == "master" {
		cluster.ObjectMeta.Annotations[TagMasterInstanceID] = instanceIDs[0]
		cluster.ObjectMeta.Annotations[TagClusterToken] = token
		machine.ObjectMeta.Annotations[TagInstanceRole] = "master"
	} else {
		machine.ObjectMeta.Annotations[TagInstanceRole] = "node"
	}

	glog.V(4).Infof("new machine: %+v, annotation %+v", machine.Name, machine.Annotations)
	cce.client.Update(context.Background(), cluster)
	cce.client.Update(context.Background(), machine)

	// TODO rewrite
	go cce.postCreate(ctx, cluster, machine)
	return nil
}

func (cce *CCEClient) postCreate(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	// check instance status
	var instanceStatusErr error
	var instance *bcc.Instance
	for i := 0; i < 10; i++ {
		time.Sleep(30 * time.Second)
		instance, instanceStatusErr = cce.instanceIfExists(cluster, machine)
		if instanceStatusErr == nil && instance.Status == "Running" {
			break
		}
		glog.V(4).Infof("check instance, pass %d, status, %s, err %+v", i, instance.Status, instanceStatusErr)
	}

	if instanceStatusErr != nil {
		glog.Errorf("instanceIfExist check err: %+v", instanceStatusErr)
		return instanceStatusErr
	}
	glog.Infof("postCreate instance %s, status %s", instance.InstanceID, instance.Status)

	role := machine.ObjectMeta.Annotations[TagInstanceRole]
	adminPass := machine.ObjectMeta.Annotations[TagInstanceAdminPass]

	var startupScript string
	if role == "master" {
		startupScript = utils.MasterStartup
	} else {
		startupScript = utils.NodeStartup
		// TODO installation of node is mush more faster, check master status
		// time.Sleep(3 * time.Minute)
	}

	masterInstance, err := cce.computeService.Bcc().DescribeInstance(cluster.ObjectMeta.Annotations[TagMasterInstanceID], nil)
	if err != nil {
		return err
	}
	glog.V(4).Info("master id %s, info %+v", cluster.ObjectMeta.Annotations[TagMasterInstanceID], masterInstance)
	startupScript = strings.Replace(startupScript, "__VERSION__", machine.Spec.Versions.Kubelet, 1) // TODO controlPlane and kubelet versions can be different
	startupScript = strings.Replace(startupScript, "__SVC_CIDR__", cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0], 1)
	startupScript = strings.Replace(startupScript, "__POD_CIDR__", cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0], 1)
	startupScript = strings.Replace(startupScript, "__PUBLICIP__", instance.PublicIP, 1)
	startupScript = strings.Replace(startupScript, "__MACHINE__", instance.InstanceID, 1)
	startupScript = strings.Replace(startupScript, "__TOKEN__", cluster.ObjectMeta.Annotations[TagClusterToken], 1)
	startupScript = strings.Replace(startupScript, "__MASTER__", masterInstance.InternalIP, 1)

	res, err := utils.RemoteSSHBashScript("root", instance.PublicIP, adminPass, startupScript)
	if err != nil {
		glog.Errorf("deploy %+v", err)
		return err
	}
	glog.Infof("postCreate result: %s", res)
	return nil
}

// Delete cleans a node
func (cce *CCEClient) Delete(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	glog.V(4).Info("Delete node: %s", machine.Name)
	kubeclient, err := cce.getKubeClient(cluster)
	if err != nil {
		return err
	}

	if err := kubeclient.CoreV1().Nodes().Delete(machine.Name, &metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	glog.V(4).Infof("Release machine: %s", machine.Name)
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
	glog.V(4).Infof("Update machine: %+v", machine.Name)
	return nil
}

// GetIP returns ip of some machine
func (cce *CCEClient) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	// TODO
	return "", nil
}

// GetKubeConfig returns config of some mahine
func (cce *CCEClient) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	// TODO store some basic info in machine struct
	// masterIntance, err := cce.instanceIfExists(cluster, master)
	masterInstanceID := cluster.ObjectMeta.Annotations[TagMasterInstanceID]
	masterInstance, err := cce.computeService.Bcc().DescribeInstance(masterInstanceID, nil)
	if err != nil {
		return "", err
	}
	return utils.RemoteSSHCommand("root", masterInstance.PublicIP, "testpw123!", "cat /root/.kube/config")
}

func (cce *CCEClient) nodeIfExists(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	// check if the node exists in the cluster
	return nil
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

func (cce *CCEClient) getKubeadmToken() (string, error) {
	// TODO generate random token
	return "abcdef.0123456789abcdef", nil
}

func (cce *CCEClient) getKubeClient(cluster *clusterv1.Cluster) (kubernetes.Interface, error) {
	// TODO get master
	configContent, err := cce.GetKubeConfig(cluster, nil)
	if err != nil {
		return nil, err
	}

	tmpDir, err := ioutil.TempDir("/tmp", cluster.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("Create tmp dir failed: '%v'", err)
	}
	defer os.Remove(tmpDir)
	cfgfile, err := ioutil.TempFile(tmpDir, "config")
	if err != nil {
		return nil, fmt.Errorf("Create tmp config file failed: '%v'", err)
	}
	defer os.Remove(cfgfile.Name())
	if err := ioutil.WriteFile(cfgfile.Name(), []byte(configContent), 0644); err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", cfgfile.Name())
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func getOrNewKubeadm(params MachineActuatorParams) CCEClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
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
	cfg.Region = "hk"
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
