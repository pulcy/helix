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
}

// setupDefaults fills given flags with default value
func (flags *ControlPlane) setupDefaults(log zerolog.Logger) error {
	if err := resolveDNSNames(flags.Members); err != nil {
		return maskAny(err)
	}
	return nil
}

// ContainsHost returns true when the given address is an entry in
// the control-plane members list.
func (flags ControlPlane) ContainsHost(addr string) bool {
	for _, x := range flags.Members {
		if x == addr {
			return true
		}
	}
	return false
}
