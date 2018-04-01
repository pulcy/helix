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
	"net"
	"sync"

	"github.com/pkg/errors"
)

// Node holds the name and address of a single cluster member.
type Node struct {
	Name           string // Hostname
	Address        string // IP address
	IsControlPlane bool
}

// CreateNodes inspects all names/addresses in the given list and resolves
// everything that is not already an IP address.
func CreateNodes(nameList []string, isControlPlane bool) ([]Node, error) {
	if len(nameList) == 0 {
		return nil, nil
	}
	result := make([]Node, len(nameList))
	wg := sync.WaitGroup{}
	errorsChan := make(chan error, len(nameList))
	defer close(errorsChan)
	for i, n := range nameList {
		wg.Add(1)
		go func(i int, n string) {
			defer wg.Done()
			if net.ParseIP(n) != nil {
				// Input is an IP address
				result[i] = Node{
					Name:           "node-" + n,
					Address:        n,
					IsControlPlane: isControlPlane,
				}
			} else {
				// Try to resolve
				addrs, err := net.LookupHost(n)
				if err != nil {
					errorsChan <- maskAny(errors.Wrapf(err, "Failed to resolve '%s'", n))
				} else if len(addrs) == 0 {
					errorsChan <- maskAny(errors.Wrapf(err, "Found not addresses for '%s'", n))
				} else {
					result[i] = Node{
						Name:           n,
						Address:        addrs[0],
						IsControlPlane: isControlPlane,
					}
				}
			}
		}(i, n)
	}
	wg.Wait()
	select {
	case err := <-errorsChan:
		return nil, maskAny(err)
	default:
		return result, nil
	}
}
