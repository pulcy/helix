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

package service

import (
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/rs/zerolog"

	"github.com/pulcy/helix/util"
)

type Service interface {
	Name() string
	Prepare(deps ServiceDependencies, flags ServiceFlags, willInit bool) error
}

type ServiceIniter interface {
	Service
	Init(deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceReseter interface {
	Service
	Reset(deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceMachines interface {
	Service
	InitMachine(node Node, client util.SSHClient, deps ServiceDependencies, flags ServiceFlags) error
	ResetMachine(node Node, client util.SSHClient, deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceDependencies struct {
	Logger         zerolog.Logger
	KubernetesCA   util.CA
	ServiceAccount struct {
		Cert string
		Key  string
	}
}

type ServiceFlags struct {
	// General
	DryRun       bool
	LocalConfDir string   // Path of local directory containing configuration (like ca certificates) files.
	Members      []string // IP/hostname of all machines (no need to include control-plane members)
	nodes        []Node
	SSH          struct {
		User string
	}
	Architecture string

	// Docker images
	Images Images

	// Control plane
	ControlPlane ControlPlane

	// ETCD
	Etcd Etcd

	// Kubernetes config
	Kubernetes Kubernetes
}

// SetupDefaults fills given flags with default value
func (flags *ServiceFlags) SetupDefaults(log zerolog.Logger) error {
	nodes, err := CreateNodes(flags.Members, false)
	if err != nil {
		return maskAny(err)
	}
	flags.nodes = nodes
	if flags.Architecture == "" {
		flags.Architecture = "amd64"
	}
	if err := flags.ControlPlane.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Etcd.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Kubernetes.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Images.setupDefaults(log, flags.Architecture, flags.Kubernetes.Version); err != nil {
		return maskAny(err)
	}
	return nil
}

// AllNodes returns a list of all members, including control plane members.
func (flags ServiceFlags) AllNodes() []Node {
	m := make(map[string]Node)
	for _, x := range flags.nodes {
		m[x.Name] = x
	}
	for _, x := range flags.ControlPlane.nodes {
		m[x.Name] = x
	}
	result := make([]Node, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Run all prepare & Setup logic of the given services.
func Run(deps ServiceDependencies, flags ServiceFlags, services []Service) error {
	// Prepare local conf dir
	confDir := flags.LocalConfDir
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return maskAny(err)
	}

	// Create Kubernetes CA
	var err error
	deps.KubernetesCA, err = util.NewCA("Kubernetes CA", filepath.Join(confDir, "kubernetes-ca.crt"), filepath.Join(confDir, "kubernetes-ca.key"))
	if err != nil {
		return maskAny(err)
	}

	// Create service account certificate
	deps.ServiceAccount.Cert, deps.ServiceAccount.Key, err = util.NewServiceAccountCertificate(filepath.Join(confDir, "kubernetes-sa.pub"), filepath.Join(confDir, "kubernetes-sa.key"))
	if err != nil {
		return maskAny(err)
	}

	// Prepare all services
	for _, s := range services {
		deps.Logger.Info().Msgf("Preparing %s service", s.Name())
		if err := s.Prepare(deps, flags, true); err != nil {
			return maskAny(err)
		}
	}

	// Dial machines
	clients, nodes, err := dialMachines(deps.Logger, flags)
	if err != nil {
		return maskAny(err)
	}
	defer func() {
		for _, c := range clients {
			c.Close()
		}
	}()

	// Setup all services on all machines
	for _, s := range services {
		if initer, ok := s.(ServiceIniter); ok {
			if err := initer.Init(deps, flags); err != nil {
				return maskAny(err)
			}
		}
		if sMachine, ok := s.(ServiceMachines); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node Node) {
					defer wg.Done()
					deps.Logger.Info().Msgf("Setting up %s service on %s", s.Name(), node.Name)
					if err := sMachine.InitMachine(node, client, deps, flags); err != nil {
						errors <- maskAny(err)
					}
				}(client, nodes[i])
			}
			wg.Wait()
			select {
			case err := <-errors:
				return maskAny(err)
			default:
				// Continue
			}
		}
	}

	return nil
}

// Reset all prepare & Setup logic of the given services.
func Reset(deps ServiceDependencies, flags ServiceFlags, services []Service) error {
	// Prepare all services
	for _, s := range services {
		deps.Logger.Info().Msgf("Preparing %s service", s.Name())
		if err := s.Prepare(deps, flags, false); err != nil {
			return maskAny(err)
		}
	}

	// Dial machines
	clients, nodes, err := dialMachines(deps.Logger, flags)
	if err != nil {
		return maskAny(err)
	}
	defer func() {
		for _, c := range clients {
			c.Close()
		}
	}()

	// Reset all services on all machines
	for _, s := range services {
		if sMachine, ok := s.(ServiceMachines); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node Node) {
					defer wg.Done()
					deps.Logger.Info().Msgf("Resetting %s service on %s", s.Name(), node.Name)
					if err := sMachine.ResetMachine(node, client, deps, flags); err != nil {
						errors <- maskAny(err)
					}
				}(client, nodes[i])
			}
			wg.Wait()
			select {
			case err := <-errors:
				return maskAny(err)
			default:
				// Continue
			}
		}
		if reseter, ok := s.(ServiceReseter); ok {
			if err := reseter.Reset(deps, flags); err != nil {
				return maskAny(err)
			}
		}
	}

	return nil
}

// dialMachines opens connections to all clients.
func dialMachines(log zerolog.Logger, flags ServiceFlags) ([]util.SSHClient, []Node, error) {
	allNodes := flags.AllNodes()
	clients := make([]util.SSHClient, len(allNodes))
	wg := sync.WaitGroup{}
	errors := make(chan error, len(allNodes))
	defer close(errors)
	for i, n := range allNodes {
		wg.Add(1)
		go func(i int, n Node) {
			defer wg.Done()
			log.Info().Msgf("Dialing %s (%s)", n.Name, n.Address)
			client, err := util.DialSSH(flags.SSH.User, n.Name, n.Address, flags.DryRun)
			if err != nil {
				errors <- maskAny(err)
			} else {
				clients[i] = client
			}
		}(i, n)
	}
	wg.Wait()
	select {
	case err := <-errors:
		return nil, nil, maskAny(err)
	default:
		return clients, allNodes, nil
	}
}
