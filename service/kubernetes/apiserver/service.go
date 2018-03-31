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
	manifestPath = "/etc/kubernetes/manifests/apiserver.yaml"

	manifestFileMode = os.FileMode(0644)
	certFileMode     = os.FileMode(0644)
	keyFileMode      = os.FileMode(0600)
)

func NewService() service.Service {
	return &apiserverService{}
}

type apiserverService struct {
	component.Component
	frontProxyCA util.CA
}

func (t *apiserverService) Name() string {
	return "kube-apiserver"
}

func (t *apiserverService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	t.Component.Name = "kube-apiserver"
	/*var err error
	t.frontProxyCA, err = util.NewCA("Kubernetes FrontProxy", true)
	if err != nil {
		return maskAny(err)
	}*/
	t.frontProxyCA = deps.KubernetesCA
	return nil
}

// SetupMachine configures the machine to run apiserver.
func (t *apiserverService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Setup apiserver on this host?
	if !flags.ControlPlane.ContainsHost(client.GetHost()) {
		log.Info().Msg("No kube-apiserver on this machine")
		return nil
	}

	cfg, err := t.createConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create & Upload certificates
	if err := t.Component.UploadCertificates(client, deps); err != nil {
		return maskAny(err)
	}

	// Create & Upload front proxy certificates
	log.Info().Msgf("Uploading %s FrontProxy Certificates", t.Name())
	proxyCert, proxyKey, err := t.frontProxyCA.CreateServerCertificate(client, true)
	if err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientCAFile, []byte(t.frontProxyCA.Cert()), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientCAKeyFile, []byte(t.frontProxyCA.Key()), keyFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientCertFile, []byte(proxyCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ProxyClientKeyFile, []byte(proxyKey), keyFileMode); err != nil {
		return maskAny(err)
	}

	// Create & Upload kubelet certificates
	log.Info().Msgf("Uploading %s Kubelet Certificates", t.Name())
	kubeletCert, kubeletKey, err := deps.KubernetesCA.CreateServerCertificate(client, true)
	if err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.KubeletCertFile, []byte(kubeletCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.KubeletKeyFile, []byte(kubeletKey), keyFileMode); err != nil {
		return maskAny(err)
	}

	// Create & Upload kubeconfig
	if err := t.Component.CreateKubeConfig(client, deps, flags); err != nil {
		return maskAny(err)
	}

	// Create manifest
	log.Info().Msg("Creating kube-apiserver manifest")
	if err := createManifest(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	return nil
}

type apiserverConfig struct {
	Image                string // HyperKube docker images
	PodName              string
	PkiDir               string
	EtcdEndpoints        string
	EtcdKeyFile          string
	EtcdCertFile         string
	EtcdCAFile           string
	FeatureGates         string // Feature gates to use
	KubeConfigPath       string // Path to a kubeconfig file, specifying how to connect to the API server.
	CertFile             string // File containing x509 Certificate used for serving HTTPS (with intermediate certs, if any, concatenated after server cert).
	KeyFile              string // File containing x509 private key matching CertPath
	ClientCAFile         string // Path of --client-ca-file
	KubeletKeyFile       string
	KubeletCertFile      string
	KubeletCAFile        string
	ProxyClientKeyFile   string
	ProxyClientCertFile  string
	ProxyClientCAFile    string
	ProxyClientCAKeyFile string
}

func (t *apiserverService) createConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (apiserverConfig, error) {
	certDir := t.Component.CertDir()
	result := apiserverConfig{
		Image:                flags.Images.HyperKube,
		PodName:              "kube-apiserver-" + client.GetHost(),
		PkiDir:               t.Component.CertRootDir(),
		EtcdEndpoints:        flags.Etcd.CreateClientEndpoints(flags.ControlPlane),
		EtcdCertFile:         filepath.Join(etcd.CertsDir, etcd.ClientCertFileName),
		EtcdKeyFile:          filepath.Join(etcd.CertsDir, etcd.ClientKeyFileName),
		EtcdCAFile:           filepath.Join(etcd.CertsDir, etcd.ClientCAFileName),
		FeatureGates:         strings.Join(flags.Kubernetes.FeatureGates, ","),
		KubeConfigPath:       t.KubeConfigPath(),
		CertFile:             t.CertPath(),
		KeyFile:              t.KeyPath(),
		ClientCAFile:         t.CAPath(),
		KubeletKeyFile:       filepath.Join(certDir, "kubelet-client.key"),
		KubeletCertFile:      filepath.Join(certDir, "kubelet-client.crt"),
		KubeletCAFile:        t.CAPath(),
		ProxyClientKeyFile:   filepath.Join(certDir, "front-proxy.key"),
		ProxyClientCertFile:  filepath.Join(certDir, "front-proxy.crt"),
		ProxyClientCAFile:    filepath.Join(certDir, "front-proxy-ca.crt"),
		ProxyClientCAKeyFile: filepath.Join(certDir, "front-proxy-ca.key"),
	}

	return result, nil
}

func createManifest(client util.SSHClient, deps service.ServiceDependencies, opts apiserverConfig) error {
	deps.Logger.Info().Msgf("Creating manifest %s", manifestPath)
	if err := client.Render(deps.Logger, apiserverManifestTemplate, manifestPath, opts, manifestFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
