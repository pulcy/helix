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
	"github.com/rs/zerolog"
)

// ControlPlane configuration
type ControlPlane struct {
	Members []string // Hostnames / IP address of all nodes that form the control plane.
	nodes   []Node
}

// setupDefaults fills given flags with default value
func (flags *ControlPlane) setupDefaults(log zerolog.Logger) error {
	nodes, err := CreateNodes(flags.Members, true)
	if err != nil {
		return maskAny(err)
	}
	flags.nodes = nodes
	return nil
}

// GetAPIServerAddress returns the IP Address of the apiserver.
func (flags *ControlPlane) GetAPIServerAddress() string {
	return flags.nodes[0].Address
}
