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

package component

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	certsRootDir       = "/etc/kubernetes/pki"
	kubeConfigsRootDir = "/var/lib"

	configFileMode = os.FileMode(0600)
	certFileMode   = os.FileMode(0644)
	keyFileMode    = os.FileMode(0600)
)

// Component is a helper for a kubernetes component
type Component struct {
	Name string
}

// CertDir returns the certificate directory for this component.
func (c Component) CertDir() string {
	return filepath.Join(certsRootDir, c.Name)
}

// CAPath returns the full path of the CA certificate file for this component.
func (c Component) CAPath() string {
	return filepath.Join(c.CertDir(), "ca.crt")
}

// CertPath returns the full path of the certificate file for this component.
func (c Component) CertPath() string {
	return filepath.Join(c.CertDir(), "cert.crt")
}

// KeyPath returns the full path of the (private) key file for this component.
func (c Component) KeyPath() string {
	return filepath.Join(c.CertDir(), "cert.key")
}

// KubeConfigPath returns the full path of the kubeconfig file for this component.
func (c Component) KubeConfigPath() string {
	return filepath.Join(kubeConfigsRootDir, c.Name, "kubeconfig")
}

// CreateKubeConfig renders and uploads a kubeconfig file for this
// component on the machine indicated by the given client.
func (c Component) CreateKubeConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	opts := struct {
		Server         string
		ContextName    string
		UserName       string
		CAPath         string
		ClientCertPath string
		ClientKeyPath  string
	}{
		Server:         flags.Etcd.Members[0],
		ContextName:    c.Name,
		UserName:       c.Name,
		CAPath:         c.CAPath(),
		ClientCertPath: c.CertPath(),
		ClientKeyPath:  c.KeyPath(),
	}
	if err := client.Render(deps.Logger, kubeConfigTemplate, c.KubeConfigPath(), opts, configFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}

// UploadCertificates creates a server certificate for the component and uploads it.
func (c Component) UploadCertificates(client util.SSHClient, deps service.ServiceDependencies) error {
	log := deps.Logger
	log.Info().Msgf("Creating %s TLS Certificates", c.Name)
	cert, key, err := deps.KubernetesCA.CreateServerCertificate(client, true)
	if err != nil {
		return maskAny(err)
	}

	log.Info().Msgf("Uploading %s TLS Certificates", c.Name)
	if err := client.UpdateFile(log, c.CertPath(), []byte(cert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, c.KeyPath(), []byte(key), keyFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, c.CAPath(), []byte(deps.KubernetesCA.Cert()), certFileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
