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
	"sort"
	"sync"

	"github.com/rs/zerolog"

	"github.com/pulcy/helix/util"
)

type Service interface {
	Name() string
	Prepare(deps ServiceDependencies, flags ServiceFlags) error
	SetupMachine(client util.SSHClient, deps ServiceDependencies, flags ServiceFlags) error
}

type ServiceDependencies struct {
	Logger       zerolog.Logger
	KubernetesCA util.CA
}

type ServiceFlags struct {
	// General
	DryRun  bool
	Members []string // IP/hostname of all machines (no need to include control-plane members)
	SSH     struct {
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
	if err := resolveDNSNames(flags.Members); err != nil {
		return maskAny(err)
	}
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

// AllMembers returns a list of all members, including control plane members.
func (flags ServiceFlags) AllMembers() []string {
	m := make(map[string]struct{})
	for _, x := range flags.Members {
		m[x] = struct{}{}
	}
	for _, x := range flags.ControlPlane.Members {
		m[x] = struct{}{}
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
	// Create Kubernetes CA
	var err error
	deps.KubernetesCA, err = util.NewCA("Kubernetes", false)
	if err != nil {
		return maskAny(err)
	}

	// Prepare all services
	for _, s := range services {
		deps.Logger.Info().Msgf("Preparing %s service", s.Name())
		if err := s.Prepare(deps, flags); err != nil {
			return maskAny(err)
		}
	}

	// Dial machines
	clients, err := dialMachines(deps.Logger, flags)
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
		wg := sync.WaitGroup{}
		errors := make(chan error, len(clients))
		defer close(errors)
		for _, client := range clients {
			wg.Add(1)
			go func(client util.SSHClient) {
				defer wg.Done()
				deps.Logger.Info().Msgf("Setting up %s service on %s", s.Name(), client.GetHost())
				if err := s.SetupMachine(client, deps, flags); err != nil {
					errors <- maskAny(err)
				}
			}(client)
		}
		wg.Wait()
		select {
		case err := <-errors:
			return maskAny(err)
		default:
			// Continue
		}
	}

	return nil
}

// dialMachines opens connections to all clients.
func dialMachines(log zerolog.Logger, flags ServiceFlags) ([]util.SSHClient, error) {
	allMembers := flags.AllMembers()
	clients := make([]util.SSHClient, len(allMembers))
	wg := sync.WaitGroup{}
	errors := make(chan error, len(allMembers))
	defer close(errors)
	for i, m := range allMembers {
		wg.Add(1)
		go func(i int, m string) {
			defer wg.Done()
			log.Info().Msgf("Dialing %s", m)
			client, err := util.DialSSH(flags.SSH.User, m, flags.DryRun)
			if err != nil {
				errors <- maskAny(err)
			} else {
				clients[i] = client
			}
		}(i, m)
	}
	wg.Wait()
	select {
	case err := <-errors:
		return nil, maskAny(err)
	default:
		return clients, nil
	}
}
