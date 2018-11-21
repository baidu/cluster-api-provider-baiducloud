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

package main

import (
	"flag"

	"github.com/golang/glog"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/apis"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/cloud/baiducloud"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/controller"
	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	flag.Parse()

	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Parsing flags")

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatal(err)
	}

	glog.Info("Initilizing now")
	initStaticDeps(mgr)

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	if err := controller.AddToManager(mgr); err != nil {
		glog.Fatal(err)
	}

	glog.Info("Cluster/Machine controller started")

	// Start the Cmd
	glog.Fatal(mgr.Start(signals.SetupSignalHandler()))
}

func initStaticDeps(mgr manager.Manager) {
	var err error
	baiducloud.MachineActuator, err = baiducloud.NewMachineActuator(baiducloud.MachineActuatorParams{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetRecorder("cce-controller"),
		Scheme:        mgr.GetScheme(),
	})
	if err != nil {
		glog.Fatal(err)
	}
	glog.V(4).Infof("initStaticDeps, machine actuator: %+v", baiducloud.MachineActuator)
	clustercommon.RegisterClusterProvisioner(baiducloud.ProviderName, baiducloud.MachineActuator)
}
