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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

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
	certFileMode     = util.CertFileMode
	keyFileMode      = util.KeyFileMode
)

func NewService() service.Service {
	return &etcdService{}
}

type etcdService struct {
	initialClusterToken string
	ca                  util.CA
	isExisting          int32
}

func (t *etcdService) Name() string {
	return "etcd"
}

func (t *etcdService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	log := deps.Logger
	t.isExisting = 0
	if willInit {
		t.initialClusterToken = uniuri.New()
		log.Info().Msg("Creating ETCD CA")
		var err error
		confDir := flags.LocalConfDir
		t.ca, err = util.NewCA("ETCD", filepath.Join(confDir, "etcd-ca.crt"), filepath.Join(confDir, "etcd-ca.key"))
		if err != nil {
			return maskAny(err)
		}
	}
	return nil
}

// InitNode looks for an existing ETCD data
func (t *etcdService) InitNode(node *service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup ETCD on this host?
	if !node.IsControlPlane {
		return nil
	}

	memberDir := filepath.Join(dataDir, "member")
	result, err := client.Run(log, fmt.Sprintf("test -d %s || echo 'not'", memberDir), "", true)
	if err != nil {
		return maskAny(err)
	}
	result = strings.TrimSpace(result)
	if result != "not" {
		// member data dir exists
		atomic.StoreInt32(&t.isExisting, 1)
	}

	return nil
}

// InitMachine configures the machine to run ETCD.
func (t *etcdService) InitMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup ETCD on this host?
	if !node.IsControlPlane {
		log.Info().Msg("No ETCD on this machine")
		return nil
	}

	cfg, err := t.createEtcdConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create certificates
	log.Info().Msg("Creating ETCD Server Certificates")
	clientCert, clientKey, err := t.ca.CreateServerCertificate(node.Name, "helix", client)
	if err != nil {
		return maskAny(err)
	}
	peerCert, peerKey, err := t.ca.CreateServerCertificate(node.Name, "helix", client)
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

// ResetMachine removes ETCD from the machine.
func (t *etcdService) ResetMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	cfg, err := t.createEtcdConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Remove manifest
	if err := client.RemoveFile(log, manifestPath); err != nil {
		return maskAny(err)
	}

	// Remove certificates
	if err := client.RemoveFile(log, cfg.ClientCertFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.ClientKeyFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.ClientCAFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.PeerCertFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.PeerKeyFile); err != nil {
		return maskAny(err)
	}
	if err := client.RemoveFile(log, cfg.PeerCAFile); err != nil {
		return maskAny(err)
	}

	// Remove data dir
	if err := client.RemoveDirectory(log, cfg.DataDir); err != nil {
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

func (t *etcdService) createEtcdConfig(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (etcdConfig, error) {
	result := etcdConfig{
		Image:               flags.Images.EtcdImage(node.Architecture),
		PeerName:            node.Name,
		PodName:             "etcd-" + node.Name,
		ClusterState:        "new",
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
	if t.isExisting != 0 {
		result.ClusterState = "existing"
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
