// Copyright (c) 2017 Pulcy.
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

package kubernetes

import (
	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	kubeDNSTemplate = "templates/kubernetes/kube-dns.yaml.tmpl"
)

// createKubeDNSAddon creates the manifest containing the kubernetes Kube-dns addon.
func createKubeDNSAddon(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	if err := util.EnsureDirectoryOf(c.AddonPath(), 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.AddonPath())
	opts := struct {
		ClusterDNS    string
		ClusterDomain string
		Version       string
	}{
		ClusterDNS:    flags.Kubernetes.ClusterDNS,
		ClusterDomain: flags.Kubernetes.ClusterDomain,
		Version:       "1.11.0",
	}
	changed, err := templates.Render(deps.Logger, kubeDNSTemplate, c.AddonPath(), opts, manifestFileMode)
	return changed, maskAny(err)
}
