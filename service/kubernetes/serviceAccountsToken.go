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
	"text/template"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	serviceAccountsTokenServiceTemplate  = "templates/kubernetes/service-accounts-token.service.tmpl"
	serviceAccountsTokenServiceName      = "service-accounts-token.service"
	serviceAccountsTokenTemplateTemplate = "templates/kubernetes/service-accounts-token.template.tmpl"
)

var (
	serviceAccountsKeyPath           = certificatePath(fmt.Sprintf("%s.key", compNameKubeServiceAccounts))
	serviceAccountsTokenTemplateName = fmt.Sprintf("%s.template", compNameKubeServiceAccounts)
)

// createServiceAccountsTemplate creates the consul-template used by the k8s-certs service.
func createServiceAccountsTemplate(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	serviceAccountsTokenTemplatePath := certificatePath(serviceAccountsTokenTemplateName)
	if err := util.EnsureDirectoryOf(serviceAccountsTokenTemplatePath, 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", serviceAccountsTokenTemplatePath)
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterID string
	}{
		ClusterID: clusterID,
	}
	setDelims := func(t *template.Template) {
		t.Delims("[[", "]]")
	}
	changed, err := templates.Render(deps.Logger, serviceAccountsTokenTemplateTemplate, serviceAccountsTokenTemplatePath, opts, templateFileMode, setDelims)
	return changed, maskAny(err)
}

// createServiceAccountsService creates the k8s-service-accounts-token extraction service.
func createServiceAccountsService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	serviceAccountsTokenServicePath := servicePath(serviceAccountsTokenServiceName)
	serviceAccountsTokenTemplatePath := certificatePath(serviceAccountsTokenTemplateName)
	deps.Logger.Info("creating %s", serviceAccountsTokenServicePath)
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		VaultMonkeyImage   string
		ConsulAddress      string
		JobID              string
		TemplatePath       string
		TemplateOutputPath string
		ConfigFileName     string
		RestartCommand     string
		TokenTemplate      string
		TokenPolicy        string
		TokenRole          string
	}{
		VaultMonkeyImage:   flags.VaultMonkeyImage,
		ConsulAddress:      flags.Network.ClusterIP + ":8500",
		JobID:              jobID(clusterID, compNameKubeServiceAccounts),
		TemplatePath:       serviceAccountsTokenTemplatePath,
		TemplateOutputPath: serviceAccountsKeyPath,
		ConfigFileName:     fmt.Sprintf("%s-config.json", compNameKubeServiceAccounts),
		RestartCommand:     fmt.Sprintf("/bin/systemctl restart %s.service", compNameKubelet),
		TokenTemplate:      `{ "vault": { "token": "{{.Token}}" }}`,
		TokenPolicy:        fmt.Sprintf("secret/%s/k8s/token/%s", clusterID, compNameKubeServiceAccounts),
		TokenRole:          tokenRole(clusterID, compNameKubeServiceAccounts),
	}
	changed, err := templates.Render(deps.Logger, serviceAccountsTokenServiceTemplate, serviceAccountsTokenServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}
