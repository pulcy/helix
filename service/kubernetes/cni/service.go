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

package cni

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	ServiceName = "cni-installer"
	servicePath = "/etc/systemd/system/" + ServiceName + ".service"

	pluginsTGZPath     = "/opt/cni-plugins-v0.7.0.tgz"
	pluginsURLTemplate = "https://github.com/containernetworking/plugins/releases/download/v0.7.0/cni-plugins-%s-v0.7.0.tgz"

	serviceFileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &cniService{}
}

type cniService struct {
}

func (t *cniService) Name() string {
	return ServiceName
}

func (t *cniService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

// InitMachine configures the machine to run download hyperkube.
func (t *cniService) InitMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()
	cfg, err := t.createConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create service
	log.Info().Msg("Creating CNI Download Service")
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
func (t *cniService) ResetMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()
	cfg, err := t.createConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Stop service
	if _, err := client.Run(log, "sudo systemctl stop "+ServiceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to stop cni service")
	}
	if _, err := client.Run(log, "sudo systemctl disable "+ServiceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to disable cni service")
	}

	// Remove service
	if err := client.RemoveFile(log, servicePath); err != nil {
		return maskAny(err)
	}

	// Remove binaries
	if err := client.RemoveDirectory(log, cfg.CniBinDir); err != nil {
		return maskAny(err)
	}

	return nil
}

type config struct {
	PluginsTgzPath string
	PluginsURL     string
	CniBinDir      string
}

func (t *cniService) createConfig(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	result := config{
		PluginsTgzPath: pluginsTGZPath,
		PluginsURL:     fmt.Sprintf(pluginsURLTemplate, node.Architecture),
		CniBinDir:      "/opt/cni/bin",
	}

	return result, nil
}

func createService(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating service %s", servicePath)
	if err := client.Render(deps.Logger, cniDownloadServiceTemplate, servicePath, opts, serviceFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
