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
	"path"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	kubeControllerManagerTemplate = "templates/kubernetes/kube-controller-manager.yaml.tmpl"
)

// createKubeControllerManagerManifest creates the manifest containing the kubernetes Kube-controller-manager pod.
func createKubeControllerManagerManifest(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	if err := util.EnsureDirectoryOf(c.ManifestPath(), 0755); err != nil {
		return false, maskAny(err)
	}
	configChanged, err := createKubeConfig(deps, flags, c)
	if err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.ManifestPath())
	apiServers, err := getAPIServers(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		Image                 string
		Master                string
		KubeConfigPath        string
		ServiceClusterIPRange string
		ServiceAccountKeyPath string
		CAPath                string
		CertificatesFolder    string
	}{
		Image:                 flags.Kubernetes.KubernetesMasterImage,
		Master:                apiServers[0],
		KubeConfigPath:        c.KubeConfigPath(),
		ServiceClusterIPRange: flags.Kubernetes.ServiceClusterIPRange,
		ServiceAccountKeyPath: serviceAccountsKeyPath,
		CAPath:                c.CAPath(),
		CertificatesFolder:    path.Dir(c.CertificatePath()),
	}
	changed, err := templates.Render(deps.Logger, kubeControllerManagerTemplate, c.ManifestPath(), opts, manifestFileMode)
	return changed || configChanged, maskAny(err)
}
