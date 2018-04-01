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

package kubelet

import (
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/service/kubernetes/component"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	serviceName = "kubelet"
	servicePath = "/etc/systemd/system/" + serviceName + ".service"

	serviceFileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &kubeletService{}
}

type kubeletService struct {
	component.Component
}

func (t *kubeletService) Name() string {
	return "kubelet"
}

func (t *kubeletService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	t.Component.Name = t.Name()
	return nil
}

// SetupMachine configures the machine to run ETCD.
func (t *kubeletService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()
	cfg, err := t.createConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create & Upload certificates
	if err := t.Component.UploadCertificates("system:node:"+client.GetHost(), "system:nodes", client, deps); err != nil {
		return maskAny(err)
	}

	// Create & Upload kubeconfig
	if err := t.Component.CreateKubeConfig(client, deps, flags); err != nil {
		return maskAny(err)
	}

	// Create service
	log.Info().Msg("Creating Kubelet Service")
	if err := createService(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	// Restart service
	if _, err := client.Run(log, "sudo systemctl daemon-reload", "", false); err != nil {
		return maskAny(err)
	}
	if _, err := client.Run(log, "sudo systemctl enable "+serviceName, "", false); err != nil {
		return maskAny(err)
	}
	if _, err := client.Run(log, "sudo systemctl restart "+serviceName, "", false); err != nil {
		return maskAny(err)
	}

	return nil
}

// ResetMachine removes kubelet from the machine.
func (t *kubeletService) ResetMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Stop service
	if _, err := client.Run(log, "sudo systemctl stop "+serviceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to stop kubelet service")
	}
	if _, err := client.Run(log, "sudo systemctl disable "+serviceName, "", true); err != nil {
		log.Warn().Err(err).Msg("Failed to disable kubelet service")
	}

	// Remove service
	if err := client.RemoveFile(log, servicePath); err != nil {
		return maskAny(err)
	}

	// Remove certificates
	if err := t.Component.RemoveCertificates(client, deps); err != nil {
		return maskAny(err)
	}

	// Remove kubeconfig
	if err := t.Component.RemoveKubeConfig(client, deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

type config struct {
	KubernetesVersion string // Version number of kubernetes
	ClusterDNS        string // Comma-separated list of DNS server IP address.
	ClusterDomain     string // Domain for this cluster.
	FeatureGates      string // Feature gates to use
	KubeConfigPath    string // Path to a kubeconfig file, specifying how to connect to the API server.
	NodeLabels        string // Labels to add when registering the node in the cluster.
	CertPath          string // File containing x509 Certificate used for serving HTTPS (with intermediate certs, if any, concatenated after server cert).
	KeyPath           string // File containing x509 private key matching CertPath
	ClientCAPath      string // Path of --client-ca-file
}

func (t *kubeletService) createConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	result := config{
		KubernetesVersion: flags.Kubernetes.Version,
		ClusterDNS:        flags.Kubernetes.ClusterDNS,
		ClusterDomain:     flags.Kubernetes.ClusterDomain,
		FeatureGates:      strings.Join(flags.Kubernetes.FeatureGates, ","),
		KubeConfigPath:    t.KubeConfigPath(),
		NodeLabels:        "",
		CertPath:          t.CertPath(),
		KeyPath:           t.KeyPath(),
		ClientCAPath:      t.CACertPath(),
	}

	return result, nil
}

func createService(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating service %s", servicePath)
	if err := client.Render(deps.Logger, kubeletServiceTemplate, servicePath, opts, serviceFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
