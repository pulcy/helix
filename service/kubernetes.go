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

package service

import (
	"fmt"

	"github.com/ericchiang/k8s"
	"github.com/rs/zerolog"
)

// K8s config
type Kubernetes struct {
	Version               string
	APIServerPort         int
	ServiceClusterIPRange string
	ClusterDNS            string   // IP address of DNS server
	ClusterDomain         string   // Name of culster domain
	FeatureGates          []string // List of activated feature gates
	Metadata              string
}

const (
	defaultKubernetesVersion     = "v1.10.0"
	defaultServiceClusterIPRange = "10.71.0.0/16"
	defaultAPIServerPort         = 6443
	defaultClusterDNS            = "10.71.0.10"
	defaultClusterDomain         = "cluster.local"
)

// setupDefaults fills given flags with default value
func (flags *Kubernetes) setupDefaults(log zerolog.Logger) error {
	if flags.Version == "" {
		flags.Version = defaultKubernetesVersion
	}
	if flags.APIServerPort == 0 {
		flags.APIServerPort = defaultAPIServerPort
	}
	if flags.ServiceClusterIPRange == "" {
		flags.ServiceClusterIPRange = defaultServiceClusterIPRange
	}
	if flags.ClusterDNS == "" {
		flags.ClusterDNS = defaultClusterDNS
	}
	if flags.ClusterDomain == "" {
		flags.ClusterDomain = defaultClusterDomain
	}
	return nil
}

// NewKubernetesClient creates a client from the outside to the k8s cluster
func NewKubernetesClient(sctx *ServiceContext, deps ServiceDependencies, flags ServiceFlags) (*k8s.Client, error) {
	cert, key, err := deps.KubernetesCA.CreateTLSClientAuthCertificate("kubernetes-admin", "system:masters", nil)
	if err != nil {
		return nil, maskAny(err)
	}
	cfg := &k8s.Config{
		Clusters: []k8s.NamedCluster{
			k8s.NamedCluster{
				Name: "k8s",
				Cluster: k8s.Cluster{
					Server: fmt.Sprintf("https://%s:6443", sctx.GetAPIServer()),
					CertificateAuthorityData: []byte(deps.KubernetesCA.Cert()),
				},
			},
		},
		AuthInfos: []k8s.NamedAuthInfo{
			k8s.NamedAuthInfo{
				Name: "admin",
				AuthInfo: k8s.AuthInfo{
					ClientCertificateData: []byte(cert),
					ClientKeyData:         []byte(key),
				},
			},
		},
		Contexts: []k8s.NamedContext{
			k8s.NamedContext{
				Name: "k8s",
				Context: k8s.Context{
					Cluster:  "k8s",
					AuthInfo: "admin",
				},
			},
		},
		CurrentContext: "k8s",
	}
	client, err := k8s.NewClient(cfg)
	if err != nil {
		return nil, maskAny(err)
	}
	return client, nil
}
