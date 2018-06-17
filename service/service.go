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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/rs/zerolog"

	"github.com/pulcy/helix/util"
)

type Service interface {
	Name() string
	Prepare(sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags, willInit bool) error
}

type ServiceIniter interface {
	Service
	Init(sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceReseter interface {
	Service
	Reset(sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceMachines interface {
	Service
	InitMachine(node Node, client util.SSHClient, sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) error
	ResetMachine(node Node, client util.SSHClient, sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceNodeInitializer interface {
	Service
	InitNode(node *Node, client util.SSHClient, sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) error
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
	SSH          struct {
		User string
	}

	// Docker images
	Images Images

	// Control plane
	ControlPlane ControlPlane

	// ETCD
	Etcd Etcd

	// Kubernetes config
	Kubernetes Kubernetes
}

type ServiceContext struct {
	flags ServiceFlags
	nodes []*Node
}

// SetupDefaults fills given flags with default value
func (flags *ServiceFlags) SetupDefaults(log zerolog.Logger, isSetup bool) error {
	if err := flags.ControlPlane.setupDefaults(log, isSetup); err != nil {
		return maskAny(err)
	}
	if err := flags.Etcd.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Kubernetes.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Images.setupDefaults(log, flags.Kubernetes.Version); err != nil {
		return maskAny(err)
	}
	return nil
}

// CreateNodes creates a list of Node objects for all members.
func (flags *ServiceFlags) CreateNodes(log zerolog.Logger, isSetup bool) ([]*Node, error) {
	nodes, err := CreateNodes(flags.Members, false)
	if err != nil {
		return nil, maskAny(err)
	}
	cpNodes, err := flags.ControlPlane.createNodes(log)
	if err != nil {
		return nil, maskAny(err)
	}
	return mergeNodes(nodes, cpNodes), nil
}

// GetControlPlaneIndex returns the index of the given node in the control plan (0...)
func (c *ServiceContext) GetControlPlaneIndex(n Node) int {
	result := 0
	for _, x := range c.nodes {
		if x.IsControlPlane {
			if n.Name == x.Name || n.Address == x.Address {
				return result
			}
			result++
		}
	}
	return -1
}

// mergeNodes returns a list of all nodes in a & b, last node wins.
func mergeNodes(a, b []*Node) []*Node {
	m := make(map[string]*Node)
	for _, x := range a {
		m[x.Name] = x
	}
	for _, x := range b {
		m[x.Name] = x
	}
	result := make([]*Node, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// GetAPIServer returns the hostname or IP Address of the apiserver.
func (c *ServiceContext) GetAPIServer() string {
	if c.flags.ControlPlane.APIServerVirtualIP != "" {
		return c.flags.ControlPlane.APIServerVirtualIP
	}
	if c.flags.ControlPlane.APIServerDNSName != "" {
		return c.flags.ControlPlane.APIServerDNSName
	}
	return c.nodes[0].Address
}

// AllArchitectures returns a list of all architectures being used.
func (c *ServiceContext) AllArchitectures() []string {
	m := make(map[string]string)
	for _, x := range c.nodes {
		if x.Architecture == "" {
			panic(fmt.Sprintf("Architecture of node %s is empty", x.Name))
		}
		m[x.Architecture] = x.Architecture
	}
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// Run all prepare & Setup logic of the given services.
func Run(deps ServiceDependencies, flags ServiceFlags, services []Service) error {
	// Prepare local conf dir
	confDir := flags.LocalConfDir
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return maskAny(err)
	}

	// Prepare context
	nodes, err := flags.CreateNodes(deps.Logger, true)
	if err != nil {
		return maskAny(err)
	}
	sctx := &ServiceContext{
		flags: flags,
		nodes: nodes,
	}

	// Create Kubernetes CA
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
		if err := s.Prepare(sctx, deps, flags, true); err != nil {
			return maskAny(err)
		}
	}

	// Dial machines
	clients, err := dialMachines(deps.Logger, flags, sctx.nodes)
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
			if err := initer.Init(sctx, deps, flags); err != nil {
				return maskAny(err)
			}
		}
		if sNode, ok := s.(ServiceNodeInitializer); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node *Node) {
					defer wg.Done()
					if err := sNode.InitNode(node, client, sctx, deps, flags); err != nil {
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
		if sMachine, ok := s.(ServiceMachines); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node Node) {
					defer wg.Done()
					deps.Logger.Info().Msgf("Setting up %s service on %s", s.Name(), node.Name)
					if err := sMachine.InitMachine(node, client, sctx, deps, flags); err != nil {
						errors <- maskAny(err)
					}
				}(client, *nodes[i])
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
	// Prepare context
	nodes, err := flags.CreateNodes(deps.Logger, true)
	if err != nil {
		return maskAny(err)
	}
	sctx := &ServiceContext{
		nodes: nodes,
	}

	// Prepare all services
	for _, s := range services {
		deps.Logger.Info().Msgf("Preparing %s service", s.Name())
		if err := s.Prepare(sctx, deps, flags, false); err != nil {
			return maskAny(err)
		}
	}

	// Dial machines
	clients, err := dialMachines(deps.Logger, flags, sctx.nodes)
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
		if sNode, ok := s.(ServiceNodeInitializer); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node *Node) {
					defer wg.Done()
					if err := sNode.InitNode(node, client, sctx, deps, flags); err != nil {
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
		if sMachine, ok := s.(ServiceMachines); ok {
			wg := sync.WaitGroup{}
			errors := make(chan error, len(clients))
			defer close(errors)
			for i, client := range clients {
				wg.Add(1)
				go func(client util.SSHClient, node Node) {
					defer wg.Done()
					deps.Logger.Info().Msgf("Resetting %s service on %s", s.Name(), node.Name)
					if err := sMachine.ResetMachine(node, client, sctx, deps, flags); err != nil {
						errors <- maskAny(err)
					}
				}(client, *nodes[i])
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
			if err := reseter.Reset(sctx, deps, flags); err != nil {
				return maskAny(err)
			}
		}
	}

	return nil
}

// dialMachines opens connections to all clients.
func dialMachines(log zerolog.Logger, flags ServiceFlags, nodes []*Node) ([]util.SSHClient, error) {
	clients := make([]util.SSHClient, len(nodes))
	wg := sync.WaitGroup{}
	errors := make(chan error, len(nodes))
	defer close(errors)
	for i, n := range nodes {
		wg.Add(1)
		go func(i int, n *Node) {
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
		return nil, maskAny(err)
	default:
		return clients, nil
	}
}
