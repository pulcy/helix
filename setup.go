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
	"github.com/pulcy/helix/service/etcd"
	"github.com/pulcy/helix/service/kubernetes/apiserver"
	"github.com/pulcy/helix/service/kubernetes/ca"
	"github.com/pulcy/helix/service/kubernetes/controllermanager"
	"github.com/pulcy/helix/service/kubernetes/hyperkube"
	"github.com/pulcy/helix/service/kubernetes/kubelet"
	"github.com/pulcy/helix/service/kubernetes/scheduler"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	cmdReset = &cobra.Command{
		Use: "reset",
		Run: runReset,
	}
	setupFlags = service.ServiceFlags{}
	resetFlags = service.ServiceFlags{}

	// Create services to setup
	services = []service.Service{
		// The order of entries is relevant!
		hyperkube.NewService(),
		ca.NewService(),
		kubelet.NewService(),
		etcd.NewService(),
		apiserver.NewService(),
		scheduler.NewService(),
		controllermanager.NewService(),
	}
)

func init() {
	// Setup
	// General
	cmdSetup.Flags().BoolVar(&setupFlags.DryRun, "dry-run", true, "If set, no changes will be made")
	cmdSetup.Flags().StringSliceVar(&setupFlags.Members, "members", nil, "IP addresses (or hostnames) of normal machines (may include control-plane members)")
	cmdSetup.Flags().StringVar(&setupFlags.Architecture, "arch", "amd64", "Architecture of the machines")
	cmdSetup.Flags().StringVar(&setupFlags.SSH.User, "ssh-user", "pi", "SSH user on all machines")
	// Control plane
	cmdSetup.Flags().StringSliceVar(&setupFlags.ControlPlane.Members, "control-plane-members", nil, "IP addresses (or hostnames) of control-plane members")
	// ETCD
	cmdSetup.Flags().StringVar(&setupFlags.Etcd.ClusterState, "etcd-cluster-state", "", "State of the ETCD cluster new|existing")
	// Kubernetes
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.APIDNSName, "k8s-api-dns-name", defaultKubernetesAPIDNSName(), "Alternate name of the Kubernetes API server")
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.Metadata, "k8s-metadata", "", "Metadata list for kubelet")

	// Reset
	// General
	cmdReset.Flags().BoolVar(&resetFlags.DryRun, "dry-run", true, "If set, no changes will be made")
	cmdReset.Flags().StringSliceVar(&resetFlags.Members, "members", nil, "IP addresses (or hostnames) of normal machines (may include control-plane members)")
	cmdReset.Flags().StringVar(&resetFlags.SSH.User, "ssh-user", "pi", "SSH user on all machines")

	cmdMain.AddCommand(cmdSetup)
	cmdMain.AddCommand(cmdReset)
}

func runSetup(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := setupFlags.SetupDefaults(cliLog); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(strings.Join(setupFlags.Members, ","), "--members")
	assertArgIsSet(strings.Join(setupFlags.ControlPlane.Members, ","), "--control-plane-members")

	deps := service.ServiceDependencies{
		Logger: cliLog,
	}

	// Go for it
	if err := service.Run(deps, setupFlags, services); err != nil {
		Exitf("Setup failed: %#v\n", err)
	}
	cliLog.Info().Msg("Done")
}

func runReset(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := resetFlags.SetupDefaults(cliLog); err != nil {
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
