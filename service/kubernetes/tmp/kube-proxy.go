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
)

const (
	kubeProxyServiceTemplate = "templates/kubernetes/kube-proxy.service.tmpl"
)

// createKubeProxyService creates the file containing the kubernetes Kube-proxy service.
func createKubeProxyService(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	configChanged, err := createKubeConfig(deps, flags, c)
	if err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.ServicePath())
	apiServers, err := getAPIServers(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		Requires         []string
		After            []string
		ClusterCIDR      string
		HostnameOverride string
		KubeConfigPath   string
		Master           string
	}{
		Requires:         []string{},
		After:            []string{c.CertificatesServiceName()},
		ClusterCIDR:      flags.Weave.IPRange,
		HostnameOverride: flags.Network.ClusterIP,
		KubeConfigPath:   c.KubeConfigPath(),
		Master:           apiServers[0],
	}
	changed, err := templates.Render(deps.Logger, kubeProxyServiceTemplate, c.ServicePath(), opts, serviceFileMode)
	return changed || configChanged, maskAny(err)
}
