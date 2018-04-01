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

package apiserver

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/service/etcd"
	"github.com/pulcy/helix/service/kubernetes/component"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	manifestPath = "/etc/kubernetes/manifests/kube-apiserver.yaml"

	manifestFileMode = os.FileMode(0644)
	certFileMode     = os.FileMode(0644)
	keyFileMode      = os.FileMode(0600)
)

func NewService() service.Service {
	return &apiserverService{}
}

type apiserverService struct {
	component.Component
}

func (t *apiserverService) Name() string {
	return "kube-apiserver"
}

func (t *apiserverService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	t.Component.Name = "apiserver"
	return nil
}

// SetupMachine configures the machine to run apiserver.
func (t *apiserverService) SetupMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup apiserver on this host?
	if !node.IsControlPlane {
		log.Info().Msg("No kube-apiserver on this machine")
		return nil
	}

	cfg, err := t.createConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create & Upload certificates
	ip, _, err := net.ParseCIDR(flags.Kubernetes.ServiceClusterIPRange)
	if err != nil {
		return maskAny(err)
	}
	ip[len(ip)-1] = 1
	altNames := []string{
		"127.0.0.1",
		ip.String(),
		"kubernetes.default.svc." + flags.Kubernetes.ClusterDomain,
		"kubernetes.default.svc",
		"kubernetes.default",
		"kubernetes",
	}
	log.Info().Strs("alt-names", altNames).Msg("apiserver.crt/key")
	if err := t.Component.UploadCertificates("kubernetes", "Kubernetes API Server", client, deps, altNames...); err != nil {
		return maskAny(err)
	}

	// Create & Upload front proxy certificates
	log.Info().Msgf("Uploading %s FrontProxy Certificates", t.Name())
	proxyCert, proxyKey, err := deps.KubernetesCA.CreateServerCertificate("kubernetes", "Kubernetes Front Proxy", client)
	if err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientCertFile, []byte(proxyCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientKeyFile, []byte(proxyKey), keyFileMode); err != nil {
		return maskAny(err)
	}

	// Create & Upload apiserver-kubelet client certificate
	log.Info().Msgf("Uploading apiserver-kubelet-client Certificates", t.Name())
	kubeletCert, kubeletKey, err := deps.KubernetesCA.CreateServerCertificate("kubernetes", "system:masters", client)
	if err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.KubeletCertFile, []byte(kubeletCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.KubeletKeyFile, []byte(kubeletKey), keyFileMode); err != nil {
		return maskAny(err)
	}

	// Create manifest
	log.Info().Msg("Creating kube-apiserver manifest")
	if err := createManifest(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	return nil
}

// ResetMachine removes kube-apiserver from the machine.
func (t *apiserverService) ResetMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	cfg, err := t.createConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Remove  manifest
	if err := client.RemoveFile(log, manifestPath); err != nil {
		return maskAny(err)
	}

	// Remove certificates
	if err := t.Component.RemoveCertificates(client, deps); err != nil {
		return maskAny(err)
	}

	// Remove front proxy certificates
	if err := client.RemoveFile(log, cfg.ProxyClientCertFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.ProxyClientKeyFile); err != nil {
		return maskAny(err)
	}

	// Remove kubelet certificates
	if err := client.RemoveFile(log, cfg.KubeletCertFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.KubeletKeyFile); err != nil {
		return maskAny(err)
	}

	return nil
}

type config struct {
	Image                  string // HyperKube docker images
	PodName                string
	ServiceClusterIPRange  string
	ClusterDomain          string
	PkiDir                 string
	EtcdEndpoints          string
	EtcdKeyFile            string
	EtcdCertFile           string
	EtcdCAFile             string
	FeatureGates           string // Feature gates to use
	APIServerCertFile      string // File containing x509 Certificate used for serving HTTPS (with intermediate certs, if any, concatenated after server cert).
	APIServerKeyFile       string // File containing x509 private key matching CertPath
	ClientCAFile           string // Path of --client-ca-file
	KubeletKeyFile         string
	KubeletCertFile        string
	KubeletCAFile          string
	ProxyClientKeyFile     string
	ProxyClientCertFile    string
	ProxyClientCAFile      string
	ProxyClientCAKeyFile   string
	ServiceAccountCertFile string
}

func (t *apiserverService) createConfig(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	certDir := t.Component.CertDir()
	result := config{
		Image:                  flags.Images.HyperKube,
		PodName:                "kube-apiserver-" + node.Name,
		ServiceClusterIPRange:  flags.Kubernetes.ServiceClusterIPRange,
		ClusterDomain:          flags.Kubernetes.ClusterDomain,
		PkiDir:                 t.Component.CertDir(),
		EtcdEndpoints:          flags.Etcd.CreateClientEndpoints(flags.ControlPlane),
		EtcdCertFile:           filepath.Join(etcd.CertsDir, etcd.ClientCertFileName),
		EtcdKeyFile:            filepath.Join(etcd.CertsDir, etcd.ClientKeyFileName),
		EtcdCAFile:             filepath.Join(etcd.CertsDir, etcd.ClientCAFileName),
		FeatureGates:           strings.Join(flags.Kubernetes.FeatureGates, ","),
		APIServerCertFile:      t.CertPath(),
		APIServerKeyFile:       t.KeyPath(),
		ClientCAFile:           t.CACertPath(),
		KubeletKeyFile:         filepath.Join(certDir, "apiserver-kubelet-client.key"),
		KubeletCertFile:        filepath.Join(certDir, "apiserver-kubelet-client.crt"),
		KubeletCAFile:          t.CACertPath(),
		ProxyClientKeyFile:     filepath.Join(certDir, "front-proxy.key"),
		ProxyClientCertFile:    filepath.Join(certDir, "front-proxy.crt"),
		ProxyClientCAFile:      t.CACertPath(),
		ProxyClientCAKeyFile:   t.CAKeyPath(),
		ServiceAccountCertFile: t.SACertPath(),
	}

	return result, nil
}

func createManifest(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating manifest %s", manifestPath)
	if err := client.Render(deps.Logger, apiserverManifestTemplate, manifestPath, opts, manifestFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
