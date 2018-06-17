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
	"io/ioutil"
	"net"

	"gopkg.in/yaml.v2"
)

// ClusterSpec holds the specification for a cluster.
type ClusterSpec struct {
	// SSHUser holds the default SSH user name used to reach the nodes
	SSHUser string `json:"ssh-user,omitempty"`
	// Masters holds the node specification for all master nodes
	Masters []NodeSpec `json:"masters"`
	// Workers holds the node specification for all worker nodes
	Workers []NodeSpec `json:"workers"`
	// APIServer holds the specification for the API server
	APIServer APIServerSpec `json:"api-server"`
}

// APIServerSpec holds the specification for the API server of a cluster.
type APIServerSpec struct {
	// VirtualIP is the virtual IP address used to reach the API server
	VirtualIP string `json:"virtual-ip,omitempty"`
	// DNSName is the DNS name of the API server
	DNSName string `json:"dns-name,omitempty"`
}

// NodeSpec holds the specification for a node of the cluster.
type NodeSpec struct {
	// Name of the node
	Name string `json:"name,omitempty"`
	// IPAddress of the node
	IPAddress string `json:"ip-address,omitempty"`
	// SSHUser holds the SSH user name used to reach the node.
	// If emty, the default SSH user name is used
	SSHUser string `json:"ssh-user,omitempty"`
}

// ParseClusterSpec parses a cluster specification in a file with given path.
func ParseClusterSpec(filePath string) (ClusterSpec, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ClusterSpec{}, maskAny(err)
	}
	var spec ClusterSpec
	if err := yaml.Unmarshal(content, &spec); err != nil {
		return ClusterSpec{}, maskAny(err)
	}
	return spec, nil
}

// Validate the given spec.
func (s ClusterSpec) Validate() error {
	for i, n := range s.Masters {
		if err := n.Validate(); err != nil {
			return maskAny(fmt.Errorf("Validation failed for master %d: %s", i+1, err))
		}
	}
	for i, n := range s.Workers {
		if err := n.Validate(); err != nil {
			return maskAny(fmt.Errorf("Validation failed for worker %d: %s", i+1, err))
		}
	}
	if err := s.APIServer.Validate(); err != nil {
		return maskAny(fmt.Errorf("Validation failed api-server: %s", err))
	}
	return nil
}

// Validate the given spec.
func (s APIServerSpec) Validate() error {
	if s.VirtualIP != "" {
		if x := net.ParseIP(s.VirtualIP); x == nil {
			return maskAny(fmt.Errorf("Invalid virtual IP Address: '%s'", s.VirtualIP))
		}
	}
	return nil
}

// Validate the given spec.
func (s NodeSpec) Validate() error {
	if s.Name == "" {
		return maskAny(fmt.Errorf("Node name cannot be empty"))
	}
	if s.IPAddress != "" {
		if x := net.ParseIP(s.IPAddress); x == nil {
			return maskAny(fmt.Errorf("Invalid IP Address: '%s'", s.IPAddress))
		}
	}
	return nil
}
