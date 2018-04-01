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

package ca

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
	certFileMode = os.FileMode(0644)
	keyFileMode  = os.FileMode(0600)
)

// NewService creates a new CA service.
func NewService() service.Service {
	return &caService{}
}

// caService installs ca.crt and ca.key (only on control plane nodes)
type caService struct {
	component.Component
}

func (t *caService) Name() string {
	return "ca"
}

func (t *caService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	return nil
}

// SetupMachine configures the machine to run apiserver.
func (t *caService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Upload ca.crt
	if err := client.UpdateFile(log, t.Component.CACertPath(), []byte(deps.KubernetesCA.Cert()), certFileMode); err != nil {
		return maskAny(err)
	}
	// If part of control, plane do a bit more.
	if flags.ControlPlane.ContainsHost(client.GetHost()) {
		// Upload ca.key
		if err := client.UpdateFile(log, t.Component.CAKeyPath(), []byte(deps.KubernetesCA.Key()), keyFileMode); err != nil {
			return maskAny(err)
		}

		// Create admin.crt, admin.key
		cert, key, err := deps.KubernetesCA.CreateAdminCertificate()
		if err != nil {
			return maskAny(err)
		}
		if err := client.UpdateFile(log, t.Component.AdminCertPath(), []byte(cert), certFileMode); err != nil {
			return maskAny(err)
		}
		if err := client.UpdateFile(log, t.Component.AdminKeyPath(), []byte(key), keyFileMode); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

// ResetMachine removes CA certificates from the machine.
func (t *caService) ResetMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Remove cert dir
	if err := client.RemoveDirectory(log, t.Component.CertDir()); err != nil {
		return maskAny(err)
	}
	// Remove cert root dir
	if err := client.RemoveDirectory(log, t.Component.CertRootDir()); err != nil {
		return maskAny(err)
	}
	return nil
}
