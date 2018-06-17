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

package keepalived

import (
	"os"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/service/kubernetes/component"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	serviceName              = "keepalived.service"
	confPath                 = "/etc/keepalived/keepalived.conf"
	apiServerCheckScriptPath = "/etc/keepalived/check_apiserver.sh"

	confFileMode   = os.FileMode(0644)
	scriptFileMode = os.FileMode(0755)
)

func NewService() service.Service {
	return &keepalivedService{}
}

type keepalivedService struct {
	component.Component
}

func (t *keepalivedService) Name() string {
	return "keepalived"
}

func (t *keepalivedService) Prepare(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	t.Component.Name = "keepalived"
	return nil
}

// InitMachine configures the machine to run apiserver.
func (t *keepalivedService) InitMachine(node service.Node, client util.SSHClient, sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup controlplane on this host?
	if !node.IsControlPlane || flags.ControlPlane.APIServerVirtualIP == "" {
		log.Info().Msg("No keepalived on this machine")
		return nil
	}

	cfg, err := t.createConfig(node, client, sctx, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create & Upload keepalived.conf
	log.Info().Msgf("Uploading %s Config", t.Name())
	if err := createConfigFile(client, deps, cfg); err != nil {
		return maskAny(err)
	}
	if err := createAPIServerCheck(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	// Restart keepalived
	if _, err := client.Run(log, "sudo systemctl restart "+serviceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to restart keepalived server")
	}

	return nil
}

// ResetMachine removes keepalived from the machine.
func (t *keepalivedService) ResetMachine(node service.Node, client util.SSHClient, sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup controlplane on this host?
	if !node.IsControlPlane || flags.ControlPlane.APIServerVirtualIP == "" {
		log.Info().Msg("No keepalived on this machine")
		return nil
	}

	// Remove config & check-script file
	if err := client.RemoveFile(log, confPath); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, apiServerCheckScriptPath); err != nil {
		return maskAny(err)
	}

	// Restart keepalived
	if _, err := client.Run(log, "sudo systemctl restart "+serviceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to restart keepalived server")
	}

	return nil
}

type config struct {
	VirtualIP    string
	State        string
	Interface    string
	Priority     int
	AuthPassword string
}

func (t *keepalivedService) createConfig(node service.Node, client util.SSHClient, sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	cpIndex := sctx.GetControlPlaneIndex(node)
	state := "BACKUP"
	if cpIndex == 0 {
		state = "MASTER"
	}
	result := config{
		VirtualIP:    flags.ControlPlane.APIServerVirtualIP,
		State:        state,
		Interface:    "eth0",
		Priority:     100 - cpIndex,
		AuthPassword: "foo",
	}

	return result, nil
}

func createConfigFile(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating config %s", confPath)
	if err := client.Render(deps.Logger, keepalivedConfTemplate, confPath, opts, confFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}

func createAPIServerCheck(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating check script %s", apiServerCheckScriptPath)
	if err := client.Render(deps.Logger, checkAPIServerTemplate, apiServerCheckScriptPath, opts, scriptFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
