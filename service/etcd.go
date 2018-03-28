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
	"fmt"
	"strings"
)

// ETCD
type Etcd struct {
	ClusterState string   // Current state of the ETCD cluster
	Members      []string // IP addresses/Hostname of ETCD members
}

const (
	defaultEtcdClientPort = 2379
	defaultEtcdPeerPort   = 2380
)

// CreateClientEndpoints returns the client URLs to reach an ETCD servers.
func (flags Etcd) CreateClientEndpoints() string {
	return flags.createEndpoints(defaultEtcdClientPort)
}

// CreatePeerEndpoints returns the peer URLs to reach an ETCD servers.
func (flags Etcd) CreatePeerEndpoints() string {
	return flags.createEndpoints(defaultEtcdPeerPort)
}

// createEndpoints returns the URLs to reach an ETCD servers at the given port.
func (flags Etcd) createEndpoints(port int) string {
	endpoints := make([]string, len(flags.Members))
	for i, m := range flags.Members {
		endpoints[i] = fmt.Sprintf("https://%s:%d", m, port)
	}
	return strings.Join(endpoints, ",")
}
