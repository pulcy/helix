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
	"encoding/base64"
	"fmt"
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
	certsDir           = "/etc/kubernetes/pki"
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
	return certsDir
}

// CACertPath returns the full path of the CA certificate file for this component.
func (c Component) CACertPath() string {
	return filepath.Join(c.CertDir(), "ca.crt")
}

// CAKeyPath returns the full path of the CA private key file for this component.
func (c Component) CAKeyPath() string {
	return filepath.Join(c.CertDir(), "ca.key")
}

// CertPath returns the full path of the certificate file for this component.
func (c Component) CertPath() string {
	return filepath.Join(c.CertDir(), c.Name+".crt")
}

// KeyPath returns the full path of the (private) key file for this component.
func (c Component) KeyPath() string {
	return filepath.Join(c.CertDir(), c.Name+".key")
}

// SACertPath returns the full path of the service account certificate file.
func (c Component) SACertPath() string {
	return filepath.Join(c.CertDir(), "sa.pub")
}

// SAKeyPath returns the full path of the service account private key file.
func (c Component) SAKeyPath() string {
	return filepath.Join(c.CertDir(), "sa.key")
}

// KubeConfigPath returns the full path of the kubeconfig file for this component.
func (c Component) KubeConfigPath() string {
	return filepath.Join(kubeConfigsRootDir, c.Name+".conf")
}

// CreateKubeConfig renders and uploads a kubeconfig file for this
// component on the machine indicated by the given client.
func (c Component) CreateKubeConfig(commonName, orgName string, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	cert, key, err := deps.KubernetesCA.CreateServerCertificate(commonName, orgName, client)
	if err != nil {
		return maskAny(err)
	}
	opts := struct {
		Server         string
		ContextName    string
		UserName       string
		CAData         string
		ClientCertData string
		ClientKeyData  string
	}{
		Server:         fmt.Sprintf("https://%s:6443", flags.ControlPlane.GetAPIServerAddress()),
		ContextName:    c.Name,
		UserName:       c.Name,
		CAData:         base64.StdEncoding.EncodeToString([]byte(deps.KubernetesCA.Cert())),
		ClientCertData: base64.StdEncoding.EncodeToString([]byte(cert)),
		ClientKeyData:  base64.StdEncoding.EncodeToString([]byte(key)),
	}
	if err := client.Render(deps.Logger, kubeConfigTemplate, c.KubeConfigPath(), opts, configFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}

// RemoveKubeConfig removes the kubeconfig file for this
// component on the machine indicated by the given client.
func (c Component) RemoveKubeConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	if err := client.RemoveFile(deps.Logger, c.KubeConfigPath()); err != nil {
		return maskAny(err)
	}
	return nil
}

// UploadCertificates creates a server certificate for the component and uploads it.
func (c Component) UploadCertificates(commonName, orgName string, client util.SSHClient, deps service.ServiceDependencies, additionalHosts ...string) error {
	log := deps.Logger
	log.Info().Msgf("Creating %s TLS Certificates", c.Name)
	cert, key, err := deps.KubernetesCA.CreateServerCertificate(commonName, orgName, client, additionalHosts...)
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

	return nil
}

// RemoveCertificates removes certificates for the component.
func (c Component) RemoveCertificates(client util.SSHClient, deps service.ServiceDependencies) error {
	if err := client.RemoveFile(deps.Logger, c.CertPath()); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(deps.Logger, c.KeyPath()); err != nil {
		return maskAny(err)
	}
	return nil
}
