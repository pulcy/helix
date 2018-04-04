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

	"github.com/rs/zerolog"
)

// Etcd holds ETCD configuration settings
type Etcd struct {
}

const (
	defaultEtcdClientPort = 2379
	defaultEtcdPeerPort   = 2380
)

// setupDefaults fills given flags with default value
func (flags *Etcd) setupDefaults(log zerolog.Logger) error {
	return nil
}

// CreateClientEndpoints returns the client URLs to reach an ETCD servers.
func (flags Etcd) CreateClientEndpoints(sctx *ServiceContext) string {
	return flags.createEndpoints(sctx, defaultEtcdClientPort, nil)
}

// CreateInitialCluster returns the peer URLs to reach an ETCD servers
// in the format accepted by --initial-cluster
func (flags Etcd) CreateInitialCluster(sctx *ServiceContext) string {
	return flags.createEndpoints(sctx, defaultEtcdPeerPort, func(node Node) string {
		return node.Name + "="
	})
}

// createEndpoints returns the URLs to reach an ETCD servers at the given port.
func (flags Etcd) createEndpoints(sctx *ServiceContext, port int, prefixBuilder func(Node) string) string {
	var endpoints []string
	for _, n := range sctx.nodes {
		if n.IsControlPlane {
			prefix := ""
			if prefixBuilder != nil {
				prefix = prefixBuilder(*n)
			}
			endpoints = append(endpoints, fmt.Sprintf("%shttps://%s:%d", prefix, n.Address, port))
		}
	}
	return strings.Join(endpoints, ",")
}
