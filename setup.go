// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/service/architecture"
	"github.com/pulcy/helix/service/etcd"
	"github.com/pulcy/helix/service/kubernetes/apiserver"
	"github.com/pulcy/helix/service/kubernetes/ca"
	"github.com/pulcy/helix/service/kubernetes/cni"
	"github.com/pulcy/helix/service/kubernetes/controllermanager"
	"github.com/pulcy/helix/service/kubernetes/controlplane"
	"github.com/pulcy/helix/service/kubernetes/coredns"
	"github.com/pulcy/helix/service/kubernetes/flannel"
	"github.com/pulcy/helix/service/kubernetes/hyperkube"
	"github.com/pulcy/helix/service/kubernetes/keepalived"
	"github.com/pulcy/helix/service/kubernetes/kubelet"
	"github.com/pulcy/helix/service/kubernetes/proxy"
	"github.com/pulcy/helix/service/kubernetes/scheduler"
)

var (
	cmdInit = &cobra.Command{
		Use: "init",
		Run: runInit,
	}
	cmdReset = &cobra.Command{
		Use: "reset",
		Run: runReset,
	}
	initFlags  = service.ServiceFlags{}
	resetFlags = service.ServiceFlags{}

	// Create services to setup
	boostrapServices = []service.Service{
		// The order of entries is relevant!
		architecture.NewService(),
		cni.NewService(),
		hyperkube.NewService(),
		keepalived.NewService(),
		ca.NewService(),
		kubelet.NewService(),
		etcd.NewService(),
		apiserver.NewService(),
		scheduler.NewService(),
		controllermanager.NewService(),
	}
	k8sServices = []service.Service{
		// The order of entries is relevant!
		architecture.NewService(),
		controlplane.NewService(),
		proxy.NewService(),
		flannel.NewService(),
		coredns.NewService(),
	}
	//	services = k8sServices // append(bootstrapServices, k8sServices...)
	services = append(boostrapServices, k8sServices...)
)

func init() {
	// cmdInit
	f := cmdInit.Flags()
	// General
	f.StringVarP(&initFlags.LocalConfDir, "conf-dir", "c", "", "Local directory containing cluster configuration")
	f.BoolVar(&initFlags.DryRun, "dry-run", false, "If set, no changes will be made")
	f.StringSliceVarP(&initFlags.Members, "members", "m", nil, "IP addresses (or hostnames) of normal machines (may include control-plane members)")
	f.StringVar(&initFlags.SSH.User, "ssh-user", "pi", "SSH user on all machines")
	// Control plane
	f.StringVar(&initFlags.ControlPlane.APIServerVirtualIP, "apiserver-virtual-ip", "", "Virtual IP address of apiserver")
	f.StringVar(&initFlags.ControlPlane.APIServerDNSName, "apiserver-dns-name", "", "DNS name of apiserver")
	f.StringSliceVar(&initFlags.ControlPlane.Members, "control-plane-members", nil, "IP addresses (or hostnames) of control-plane members")
	// Kubernetes
	f.StringVar(&initFlags.Kubernetes.Metadata, "k8s-metadata", "", "Metadata list for kubelet")

	// cmdReset
	f = cmdReset.Flags()
	// General
	f.BoolVar(&resetFlags.DryRun, "dry-run", false, "If set, no changes will be made")
	f.StringSliceVar(&resetFlags.Members, "members", nil, "IP addresses (or hostnames) of normal machines (may include control-plane members)")
	f.StringVar(&resetFlags.SSH.User, "ssh-user", "pi", "SSH user on all machines")

	cmdMain.AddCommand(cmdInit)
	cmdMain.AddCommand(cmdReset)
}

func runInit(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := initFlags.SetupDefaults(cliLog, true); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(initFlags.LocalConfDir, "--conf-dir")
	assertArgIsSet(strings.Join(initFlags.Members, ","), "--members")

	deps := service.ServiceDependencies{
		Logger: cliLog,
	}

	// Go for it
	if err := service.Run(deps, initFlags, services); err != nil {
		Exitf("Setup failed: %#v\n", err)
	}
	cliLog.Info().Msg("Done")
}

func runReset(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := resetFlags.SetupDefaults(cliLog, false); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(strings.Join(resetFlags.Members, ","), "--members")

	deps := service.ServiceDependencies{
		Logger: cliLog,
	}

	revServices := make([]service.Service, len(services))
	for i, s := range services {
		revServices[len(services)-(1+i)] = s
	}

	// Go for it
	if err := service.Reset(deps, resetFlags, revServices); err != nil {
		Exitf("Reset failed: %#v\n", err)
	}
	cliLog.Info().Msg("Done")
}
