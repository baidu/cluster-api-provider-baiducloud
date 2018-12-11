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
	"os"

	"github.com/golang/glog"

	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/apis"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/cloud/baiducloud"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/controller"
	"sigs.k8s.io/cluster-api-provider-baiducloud/pkg/webhook"

	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/controller/machinedeployment"
	"sigs.k8s.io/cluster-api/pkg/controller/machineset"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	// Get a config to talk to the apiserver
	glog.Info("setting up client for manager")
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Error(err, "unable to set up client config")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	glog.Info("setting up manager")
	flag.Parse()
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	glog.Info("Registering Components.")
	initStaticDeps(mgr)

	// Setup Scheme for all resources
	glog.Info("setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Error(err, "unable add APIs to scheme")
		os.Exit(1)
	}

	if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Error(err, "unable add cluster APIs to scheme")
		os.Exit(1)
	}

	// Setup all Controllers
	glog.Info("Setting up controller")
	if err := controller.AddToManager(mgr); err != nil {
		glog.Error(err, "unable to register controllers to the manager")
		os.Exit(1)
	}

	glog.Info("Setting up machineset")
	if err := machineset.Add(mgr); err != nil {
		glog.Error(err, "unable to register machineset to the manager")
		os.Exit(1)
	}

	glog.Info("Setting up machinedeloyment")
	if err := machinedeployment.Add(mgr); err != nil {
		glog.Error(err, "unable to register machinedeployment to the manager")
		os.Exit(1)
	}

	glog.Info("setting up webhooks")
	if err := webhook.AddToManager(mgr); err != nil {
		glog.Error(err, "unable to register webhooks to the manager")
		os.Exit(1)
	}

	// Start the Cmd
	glog.Info("Starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Error(err, "unable to run the manager")
		os.Exit(1)
	}
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
