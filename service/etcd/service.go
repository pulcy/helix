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

package etcd

import (
	"os"
	"path/filepath"

	"github.com/dchest/uniuri"
	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	manifestPath       = "/etc/kubernetes/manifests/etcd.yaml"
	CertsDir           = "/etc/kubernetes/pki/etcd"
	dataDir            = "/var/lib/etcd"
	ClientCertFileName = "client.crt"
	ClientKeyFileName  = "client.key"
	ClientCAFileName   = "ca.crt"
	peerCertFileName   = "peer.crt"
	peerKeyFileName    = "peer.key"
	peerCAFileName     = "peer-ca.crt"

	manifestFileMode = os.FileMode(0644)
	certFileMode     = os.FileMode(0644)
	keyFileMode      = os.FileMode(0600)
	dataPathMode     = os.FileMode(0755)
	initFileMode     = os.FileMode(0755)
)

func NewService() service.Service {
	return &etcdService{}
}

type etcdService struct {
	initialClusterToken string
	ca                  util.CA
}

func (t *etcdService) Name() string {
	return "etcd"
}

func (t *etcdService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger
	t.initialClusterToken = uniuri.New()
	log.Info().Msg("Creating ETCD CA")
	var err error
	t.ca, err = util.NewCA("ETCD", false)
	if err != nil {
		return maskAny(err)
	}
	return nil
}

// SetupMachine configures the machine to run ETCD.
func (t *etcdService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Setup ETCD on this host?
	if !flags.ControlPlane.ContainsHost(client.GetHost()) {
		log.Info().Msg("No ETCD on this machine")
		return nil
	}

	cfg, err := t.createEtcdConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create certificates
	log.Info().Msg("Creating ETCD Server Certificates")
	clientCert, clientKey, err := t.ca.CreateServerCertificate(client, true)
	if err != nil {
		return maskAny(err)
	}
	peerCert, peerKey, err := t.ca.CreateServerCertificate(client, true)
	if err != nil {
		return maskAny(err)
	}

	// Upload certificates
	log.Info().Msg("Uploading ETCD Server Certificates")
	if err := client.UpdateFile(log, cfg.ClientCertFile, []byte(clientCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ClientKeyFile, []byte(clientKey), keyFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.ClientCAFile, []byte(t.ca.Cert()), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerCertFile, []byte(peerCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerKeyFile, []byte(peerKey), keyFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerCAFile, []byte(t.ca.Cert()), certFileMode); err != nil {
		return maskAny(err)
	}

	// Create manifest
	log.Info().Msg("Creating ETCD Manifest")
	if err := createManifest(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	return nil
}

type etcdConfig struct {
	Image               string
	PeerName            string
	PodName             string
	ClusterState        string
	InitialCluster      string
	InitialClusterToken string
	CertificatesDir     string // Directory containing certificates
	DataDir             string // Directory containing ETCD data
	ClientCertFile      string // Path of --cert-file
	ClientKeyFile       string // Path of --key-file
	ClientCAFile        string // Path of --trusted-ca-file
	PeerCertFile        string // Path of --peer-cert-file
	PeerKeyFile         string // Path of --peer-key-file
	PeerCAFile          string // Path of --peer-trusted-ca-file
}

func (t *etcdService) createEtcdConfig(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (etcdConfig, error) {
	result := etcdConfig{
		Image:               flags.Images.Etcd,
		PeerName:            client.GetHost(),
		PodName:             "etcd-" + client.GetHost(),
		ClusterState:        flags.Etcd.ClusterState,
		InitialCluster:      flags.Etcd.CreateInitialCluster(flags.ControlPlane),
		InitialClusterToken: t.initialClusterToken,
		CertificatesDir:     CertsDir,
		DataDir:             dataDir,
		ClientCertFile:      filepath.Join(CertsDir, ClientCertFileName),
		ClientKeyFile:       filepath.Join(CertsDir, ClientKeyFileName),
		ClientCAFile:        filepath.Join(CertsDir, ClientCAFileName),
		PeerCertFile:        filepath.Join(CertsDir, peerCertFileName),
		PeerKeyFile:         filepath.Join(CertsDir, peerKeyFileName),
		PeerCAFile:          filepath.Join(CertsDir, peerCAFileName),
	}
	if result.ClusterState == "" {
		result.ClusterState = "new"
	}

	return result, nil
}

func createManifest(client util.SSHClient, deps service.ServiceDependencies, opts etcdConfig) error {
	deps.Logger.Info().Msgf("Creating manifest %s", manifestPath)
	if err := client.Render(deps.Logger, etcdManifestTemplate, manifestPath, opts, manifestFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
