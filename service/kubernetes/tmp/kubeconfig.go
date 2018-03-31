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
	"fmt"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	kubeConfigTemplate = "templates/kubernetes/kubeconfig.tmpl"
)

// createKubeConfig creates a component specific kubeconfig configuration file.
func createKubeConfig(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	deps.Logger.Info("creating %s", c.KubeConfigPath())
	if err := util.EnsureDirectoryOf(c.KubeConfigPath(), 0755); err != nil {
		return false, maskAny(err)
	}
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	var apiServers []string
	for _, m := range members {
		if !m.EtcdProxy {
			apiServers = append(apiServers, fmt.Sprintf("https://%s:%d", m.ClusterIP, flags.Kubernetes.APIServerPort))
		}
	}
	opts := struct {
		Server         string
		ContextName    string
		UserName       string
		CAPath         string
		ClientCertPath string
		ClientKeyPath  string
	}{
		Server:         apiServers[0],
		ContextName:    c.Name(),
		UserName:       c.Name(),
		CAPath:         c.CAPath(),
		ClientCertPath: c.CertificatePath(),
		ClientKeyPath:  c.KeyPath(),
	}
	changed, err := templates.Render(deps.Logger, kubeConfigTemplate, c.KubeConfigPath(), opts, configFileMode)
	return changed, maskAny(err)
}
