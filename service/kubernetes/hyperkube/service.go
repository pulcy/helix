// Copyright (c) 2018 Pulcy.
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

package hyperkube

import (
	"os"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	ServiceName = "hyperkube"
	servicePath = "/etc/systemd/system/" + ServiceName + ".service"

	serviceFileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &hyperkubeService{}
}

type hyperkubeService struct {
}

func (t *hyperkubeService) Name() string {
	return ServiceName
}

func (t *hyperkubeService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	return nil
}

// SetupMachine configures the machine to run download hyperkube.
func (t *hyperkubeService) SetupMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()
	cfg, err := t.createConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create service
	log.Info().Msg("Creating Hyperkube Service")
	if err := createService(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	// Restart service
	if _, err := client.Run(log, "sudo systemctl daemon-reload", "", false); err != nil {
		return maskAny(err)
	}
	if _, err := client.Run(log, "sudo systemctl enable "+ServiceName, "", false); err != nil {
		return maskAny(err)
	}
	if _, err := client.Run(log, "sudo systemctl restart "+ServiceName, "", false); err != nil {
		return maskAny(err)
	}

	return nil
}

// ResetMachine removes hyperkube from the machine.
func (t *hyperkubeService) ResetMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()
	cfg, err := t.createConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Stop service
	if _, err := client.Run(log, "sudo systemctl stop "+ServiceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to stop hyperkube server")
	}
	if _, err := client.Run(log, "sudo systemctl disable "+ServiceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to disable hyperkube server")
	}

	// Remove service
	if err := client.RemoveFile(log, servicePath); err != nil {
		return maskAny(err)
	}

	// Remove binaries
	if err := client.RemoveFile(log, cfg.KubeCtlPath); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.HyperKubePath); err != nil {
		return maskAny(err)
	}

	return nil
}

type config struct {
	Image             string // HyperKube docker images
	KubernetesVersion string // Version number of kubernetes
	HyperKubePath     string
	KubeCtlPath       string
}

func (t *hyperkubeService) createConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	result := config{
		Image:             flags.Images.HyperKube,
		KubernetesVersion: flags.Kubernetes.Version,
		HyperKubePath:     "/usr/local/bin/hyperkube-" + flags.Kubernetes.Version,
		KubeCtlPath:       "/usr/local/bin/kubectl",
	}

	return result, nil
}

func createService(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating service %s", servicePath)
	if err := client.Render(deps.Logger, hyperkubeServiceTemplate, servicePath, opts, serviceFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
