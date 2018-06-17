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
	"net"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// ControlPlane configuration
type ControlPlane struct {
	APIServerVirtualIP string   // Virtual IP address of APIServer
	APIServerDNSName   string   // DNS name of APIServer
	Members            []string // Hostnames / IP address of all nodes that form the control plane.
}

// setupDefaults fills given flags with default value
func (flags *ControlPlane) setupDefaults(log zerolog.Logger, isSetup bool) error {
	if len(flags.Members) == 0 && flags.APIServerVirtualIP == "" && flags.APIServerDNSName == "" && isSetup {
		return maskAny(fmt.Errorf("No control-plane members specified"))
	}
	return nil
}

// setupDefaults fills given flags with default value
func (flags *ControlPlane) createNodes(log zerolog.Logger) ([]*Node, error) {
	if len(flags.Members) > 0 {
		nodes, err := CreateNodes(flags.Members, true)
		if err != nil {
			return nil, maskAny(err)
		}
		return nodes, nil
	}
	if flags.APIServerVirtualIP == "" && flags.APIServerDNSName != "" {
		// Resolve IP address of APIServer to fetch control plane members
		addrs, err := net.LookupHost(flags.APIServerDNSName)
		if err != nil {
			return nil, maskAny(err)
		}
		nodes := make([]*Node, len(addrs))
		errors := make(chan error, len(addrs))
		defer close(errors)
		wg := sync.WaitGroup{}
		for i, addr := range addrs {
			wg.Add(1)
			go func(i int, addr string) {
				defer wg.Done()
				names, err := net.LookupAddr(addr)
				if err != nil {
					errors <- maskAny(err)
				} else if len(names) == 0 {
					errors <- maskAny(fmt.Errorf("No names found for address %s", addr))
				} else {
					nameParts := strings.Split(names[0], ".")
					nodes[i] = &Node{
						Name:           nameParts[0],
						Address:        addr,
						IsControlPlane: true,
					}
					log.Info().Msgf("Found control-plane member %s with address %s", nodes[i].Name, nodes[i].Address)
				}
			}(i, addr)
		}
		wg.Wait()
		select {
		case err := <-errors:
			return nil, maskAny(err)
		default:
			return nodes, nil
		}
	}
	return nil, nil
}
