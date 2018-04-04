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
	CoreDNSVersion string
	EtcdVersion    string
	FlannelVersion string

	k8sVersion string
}

const (
	defaultEtcdVersion    = "3.2.17"
	defaultFlannelVersion = "v0.9.1"
	defaultCoreDNSVersion = "1.1.1"

	etcdImageTemplate      = "gcr.io/google-containers/etcd-%s:%s"
	flannelImageTemplate   = "quay.io/coreos/flannel:%s-%s"
	hyperKubeImageTemplate = "gcr.io/google-containers/hyperkube-%s:%s"
	coreDNSImageTemplate   = "coredns/coredns:%s"
)

// setupDefaults fills given flags with default value
func (flags *Images) setupDefaults(log zerolog.Logger, k8sVersion string) error {
	flags.k8sVersion = k8sVersion
	if flags.CoreDNSVersion == "" {
		flags.CoreDNSVersion = defaultCoreDNSVersion
	}
	if flags.EtcdVersion == "" {
		flags.EtcdVersion = defaultEtcdVersion
	}
	if flags.FlannelVersion == "" {
		flags.FlannelVersion = defaultFlannelVersion
	}
	return nil
}

// CoreDNSImage returns the CoreDNS image name.
func (flags Images) CoreDNSImage() string {
	return fmt.Sprintf(coreDNSImageTemplate, flags.CoreDNSVersion)
}

// EtcdImage returns the ETCD image name.
func (flags Images) EtcdImage(architecture string) string {
	return fmt.Sprintf(etcdImageTemplate, architecture, flags.EtcdVersion)
}

// FlannelImage returns the flannel image name.
func (flags Images) FlannelImage(architecture string) string {
	return fmt.Sprintf(flannelImageTemplate, flags.FlannelVersion, architecture)
}

// HyperKubeImage returns the hyperkube image name.
func (flags Images) HyperKubeImage(architecture string) string {
	return fmt.Sprintf(hyperKubeImageTemplate, architecture, flags.k8sVersion)
}
