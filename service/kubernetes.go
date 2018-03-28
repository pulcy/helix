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

package service

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// K8s config
type Kubernetes struct {
	Enabled               bool
	KubernetesMasterImage string
	APIServerPort         int
	ServiceClusterIPRange string
	ClusterDNS            string // IP address of DNS server
	ClusterDomain         string // Name of culster domain
	APIDNSName            string
	Metadata              string
}

const (
	defaultKubernetesMasterImage = "pulcy/k8s-master:0.1.5"
	defaultServiceClusterIPRange = "10.71.0.0/16"
	defaultAPIServerPort         = 6443
	defaultClusterDNS            = "10.71.0.10"
	defaultClusterDomain         = "cluster.local"
)

const (
	kubeletMetadataPath       = "/etc/pulcy/kubelet-metadata"
	obsoleteFleetMetadataPath = "/etc/pulcy/fleet-metadata"
)

// setupDefaults fills given flags with default value
func (flags *Kubernetes) setupDefaults(log zerolog.Logger) error {
	if flags.KubernetesMasterImage == "" {
		flags.KubernetesMasterImage = defaultKubernetesMasterImage
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
	if flags.APIDNSName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return maskAny(err)
		}
		parts := strings.Split(hostname, "-")
		if len(parts) == 1 {
			parts = strings.Split(hostname, ".")
		}
		if len(parts) > 1 {
			flags.APIDNSName = strings.Join(parts[1:], ".")
		} else {
			flags.APIDNSName = hostname
		}
	}
	return nil
}

// IsEnabled returns true if kubernetes should be installed on the cluster.
func (flags *Kubernetes) IsEnabled() bool {
	return flags.Enabled
}
