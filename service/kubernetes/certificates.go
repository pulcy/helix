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
	"net"
	"path/filepath"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

const (
	certsServiceTemplate = "templates/kubernetes/certs.service.tmpl"
	certsTimerTemplate   = "templates/kubernetes/certs.timer.tmpl"
)

// createCertsService creates the k8s-certs service.
func createCertsService(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component, altNames []string, addInternalApiServerIP bool) (bool, error) {
	deps.Logger.Info("creating %s", c.CertificatesServicePath())
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	privateHostIP, err := flags.PrivateHostIP(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		VaultMonkeyImage   string
		JobID              string
		Component          string
		CommonName         string
		Role               string
		AltNames           []string
		IPSans             []string
		CertFileName       string
		KeyFileName        string
		CAFileName         string
		CertificatesFolder string
	}{
		VaultMonkeyImage:   flags.VaultMonkeyImage,
		JobID:              c.JobID(clusterID),
		Component:          c.Name(),
		CommonName:         c.Name(),
		Role:               c.Name(),
		AltNames:           altNames,
		IPSans:             []string{flags.Network.ClusterIP, privateHostIP},
		CertFileName:       filepath.Base(c.CertificatePath()),
		KeyFileName:        filepath.Base(c.KeyPath()),
		CAFileName:         filepath.Base(c.CAPath()),
		CertificatesFolder: certificatePath(""),
	}
	if addInternalApiServerIP {
		serviceIP, _, err := net.ParseCIDR(flags.Kubernetes.ServiceClusterIPRange)
		if err != nil {
			return false, maskAny(err)
		}
		internalApiServerIP := serviceIP.To4()
		internalApiServerIP[3] = 1
		opts.IPSans = append(opts.IPSans, internalApiServerIP.String())
	}
	changed, err := templates.Render(deps.Logger, certsServiceTemplate, c.CertificatesServicePath(), opts, serviceFileMode)
	return changed, maskAny(err)
}

// createCertsTimer creates the k8s-certs timer.
func createCertsTimer(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	deps.Logger.Info("creating %s", c.CertificatesTimerPath())
	opts := struct {
		Component string
	}{
		Component: c.Name(),
	}
	changed, err := templates.Render(deps.Logger, certsTimerTemplate, c.CertificatesTimerPath(), opts, serviceFileMode)
	return changed, maskAny(err)
}
