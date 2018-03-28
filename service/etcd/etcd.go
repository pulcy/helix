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
	//"github.com/pulcy/helix/templates"
)

var (
	maskAny = errors.WithStack
)

const (
	manifestPath       = "/etc/kubernetes/manifests/etcd.yaml"
	certsDir           = "/etc/kubernetes/pki/etcd"
	dataDir            = "/var/lib/etcd"
	clientCertFileName = "client.crt"
	clientKeyFileName  = "client.key"
	clientCAFileName   = "ca.crt"
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
	clientCA            ca
	peerCA              ca
}

func (t *etcdService) Name() string {
	return "etcd"
}

func (t *etcdService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger
	t.initialClusterToken = uniuri.New()
	log.Info().Msg("Creating ETCD Client CA")
	if err := t.clientCA.CreateCA("ETCD Clients", false); err != nil {
		return maskAny(err)
	}
	log.Info().Msg("Creating ETCD Peer CA")
	if err := t.peerCA.CreateCA("ETCD Peers", false); err != nil {
		return maskAny(err)
	}
	return nil
}

// SetupMachine configures the machine to run ETCD.
func (t *etcdService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", client.GetHost()).Logger()

	// Setup ETCD on this host?
	if !flags.Etcd.ContainsHost(client.GetHost()) {
		log.Info().Msg("No ETCD on this machine")
		return nil
	}

	cfg, err := t.createEtcdConfig(client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create certificates
	log.Info().Msg("Creating ETCD Server Certificates")
	clientCert, clientKey, err := t.clientCA.CreateServerCertificate(client)
	if err != nil {
		return maskAny(err)
	}
	peerCert, peerKey, err := t.clientCA.CreateServerCertificate(client)
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
	if err := client.UpdateFile(log, cfg.ClientCAFile, []byte(t.clientCA.caCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerCertFile, []byte(peerCert), certFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerKeyFile, []byte(peerKey), keyFileMode); err != nil {
		return maskAny(err)
	}
	if err := client.UpdateFile(log, cfg.PeerCAFile, []byte(t.peerCA.caCert), certFileMode); err != nil {
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
		InitialCluster:      flags.Etcd.CreateInitialCluster(),
		InitialClusterToken: t.initialClusterToken,
		CertificatesDir:     certsDir,
		DataDir:             dataDir,
		ClientCertFile:      filepath.Join(certsDir, clientCertFileName),
		ClientKeyFile:       filepath.Join(certsDir, clientKeyFileName),
		ClientCAFile:        filepath.Join(certsDir, clientCAFileName),
		PeerCertFile:        filepath.Join(certsDir, peerCertFileName),
		PeerKeyFile:         filepath.Join(certsDir, peerKeyFileName),
		PeerCAFile:          filepath.Join(certsDir, peerCAFileName),
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
