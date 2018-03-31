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
	"github.com/pulcy/helix/service/kubernetes/kubelet"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = service.ServiceFlags{}
)

func init() {
	// General
	cmdSetup.Flags().BoolVar(&setupFlags.DryRun, "dry-run", true, "If set, no changes will be made")
	cmdSetup.Flags().StringSliceVar(&setupFlags.AllMembers, "members", nil, "IP addresses (or hostnames) of machines")
	cmdSetup.Flags().StringVar(&setupFlags.Architecture, "arch", "amd64", "Architecture of the machines")
	cmdSetup.Flags().StringVar(&setupFlags.SSH.User, "ssh-user", "pi", "SSH user on all machines")
	// ETCD
	cmdSetup.Flags().StringVar(&setupFlags.Etcd.ClusterState, "etcd-cluster-state", "", "State of the ETCD cluster new|existing")
	cmdSetup.Flags().StringSliceVar(&setupFlags.Etcd.Members, "etcd-members", nil, "IP addresses (or hostnames) of ETCD members")
	// Kubernetes
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.APIDNSName, "k8s-api-dns-name", defaultKubernetesAPIDNSName(), "Alternate name of the Kubernetes API server")
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.Metadata, "k8s-metadata", "", "Metadata list for kubelet")

	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := setupFlags.SetupDefaults(cliLog); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(strings.Join(setupFlags.AllMembers, ","), "--members")
	assertArgIsSet(strings.Join(setupFlags.Etcd.Members, ","), "--etcd-members")

	deps := service.ServiceDependencies{
		Logger: cliLog,
	}

	// Create services to setup
	services := []service.Service{
		// The order of entries is relevant!
		kubelet.NewService(),
		etcd.NewService(),
		//		kubernetes.NewService(),
	}

	// Go for it
	if err := service.Run(deps, setupFlags, services); err != nil {
		Exitf("Setup failed: %#v\n", err)
	}
	cliLog.Info().Msg("Done")
}
