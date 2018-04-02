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

	"github.com/rs/zerolog"
)

// Images holds docker image names
type Images struct {
	Etcd      string
	HyperKube string
	CoreDNS   string
}

const (
	etcdImageTemplate      = "gcr.io/google-containers/etcd-%s:3.2.17"
	hyperKubeImageTemplate = "gcr.io/google-containers/hyperkube-%s:%s"
	defaultCoreDNSImage    = "coredns/coredns:1.1.1"
)

// setupDefaults fills given flags with default value
func (flags *Images) setupDefaults(log zerolog.Logger, architecture, k8sVersion string) error {
	if flags.Etcd == "" {
		flags.Etcd = fmt.Sprintf(etcdImageTemplate, architecture)
	}
	if flags.HyperKube == "" {
		flags.HyperKube = fmt.Sprintf(hyperKubeImageTemplate, architecture, k8sVersion)
	}
	if flags.CoreDNS == "" {
		flags.CoreDNS = defaultCoreDNSImage
	}
	return nil
}
