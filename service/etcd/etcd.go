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

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
	//"github.com/pulcy/helix/templates"
)

var (
	maskAny = errors.WithStack
)

const (
	manifestPath  = "/etc/kubernetes/manifests/etcd.yaml"
	CertsCertPath = "/opt/certs/etcd-cert.pem"
	CertsKeyPath  = "/opt/certs/etcd-key.pem"
	CertsCAPath   = "/opt/certs/etcd-ca.pem"

	manifestFileMode = os.FileMode(0644)
	dataPathMode     = os.FileMode(0755)
	initFileMode     = os.FileMode(0755)
)

func NewService() service.Service {
	return &etcdService{}
}

type etcdService struct{}

func (t *etcdService) Name() string {
	return "etcd"
}

func (t *etcdService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	return nil
}

func (t *etcdService) SetupMachine(client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	cfg, err := createEtcdConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if err := createManifest(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	return nil
}

type etcdConfig struct {
	Image               string
	PodName             string
	ClusterState        string
	InitialCluster      string
	InitialClusterToken string
}

func createEtcdConfig(deps service.ServiceDependencies, flags service.ServiceFlags) (etcdConfig, error) {
	result := etcdConfig{
		Image:               flags.Images.Etcd,
		PodName:             "etcd",
		ClusterState:        flags.Etcd.ClusterState,
		InitialCluster:      flags.Etcd.CreatePeerEndpoints(),
		InitialClusterToken: "foo",
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
