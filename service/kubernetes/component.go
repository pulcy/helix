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

import "fmt"

type Component struct {
	name       string
	masterOnly bool
	isManifest bool
	hasTimer   bool
}

// NewServiceComponent creates a new component that runs in a systemd service
func NewServiceComponent(name string, masterOnly bool) Component {
	return Component{name, masterOnly, false, false}
}

// NewServiceAndTimerComponent creates a new component that runs in a systemd service & timer
func NewServiceAndTimerComponent(name string, masterOnly bool) Component {
	return Component{name, masterOnly, false, true}
}

// NewManifestComponent creates a new component that runs in a static pod inside kubelet
func NewManifestComponent(name string, masterOnly bool) Component {
	return Component{name, masterOnly, true, false}
}

// String returns the name of the component
func (c Component) String() string {
	return c.name
}

// Name returns the name of the component
func (c Component) Name() string {
	return c.name
}

// MasterOnly returns true if this component should only be deployed on master nodes.
func (c Component) MasterOnly() bool {
	return c.masterOnly
}

// IsManifest returns true if this component is deployed as a static pod inside kubelet using a manifest.
func (c Component) IsManifest() bool {
	return c.isManifest
}

// HasTimer returns true if this component is deployed as a systemd service + timer.
func (c Component) HasTimer() bool {
	return c.hasTimer
}

// ServiceName returns the name of the systemd service that runs the component.
func (c Component) ServiceName() string {
	if c.isManifest {
		return ""
	}
	return fmt.Sprintf("%s.service", c)
}

// ServicePath returns the full path of the file containing the systemd service that runs the component.
func (c Component) ServicePath() string {
	if c.isManifest {
		return ""
	}
	return servicePath(c.ServiceName())
}

// TimerName returns the name of the systemd timer that runs the component.
func (c Component) TimerName() string {
	if c.isManifest || !c.hasTimer {
		return ""
	}
	return fmt.Sprintf("%s.timer", c)
}

// TimerPath returns the full path of the file containing the systemd timer that runs the component.
func (c Component) TimerPath() string {
	if c.isManifest || !c.hasTimer {
		return ""
	}
	return servicePath(c.TimerName())
}

// ManifestName returns the name of the static pod manifest that runs the component.
func (c Component) ManifestName() string {
	if !c.isManifest {
		return ""
	}
	return fmt.Sprintf("%s.yaml", c)
}

// ManifestPath returns the full path of the file containing the static pod manifest that runs the component.
func (c Component) ManifestPath() string {
	if !c.isManifest {
		return ""
	}
	return manifestPath(c.ManifestName())
}

// AddonPath returns the full path of the file containing the addon that runs the component.
func (c Component) AddonPath() string {
	if !c.isManifest {
		return ""
	}
	return addonPath(c.ManifestName())
}

// CertificatesServiceName returns the name of the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServiceName() string {
	return fmt.Sprintf("%s-certs.service", c)
}

// CertificatesServicePath returns the full path of the file containing the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServicePath() string {
	return servicePath(c.CertificatesServiceName())
}

// CertificatesTimerName returns the name of the systemd timer that generates the TLS certificates for the component.
func (c Component) CertificatesTimerName() string {
	return fmt.Sprintf("%s-certs.timer", c)
}

// CertificatesTimerPath returns the full path of the file containing the systemd timer that generates the TLS certificates for the component.
func (c Component) CertificatesTimerPath() string {
	return servicePath(c.CertificatesTimerName())
}

// CertificatePath returns the full path of the public key part of the certificate for this component.
func (c Component) CertificatePath() string {
	return fmt.Sprintf("/opt/certs/%s-cert.pem", c)
}

// KeyPath returns the full path of the private key part of the certificate for this component.
func (c Component) KeyPath() string {
	return fmt.Sprintf("/opt/certs/%s-key.pem", c)
}

// CAPath returns the full path of the CA certificate for this component.
func (c Component) CAPath() string {
	return fmt.Sprintf("/opt/certs/%s-ca.pem", c)
}

// KubeConfigPath returns the full path of the kubeconfig configuration file for this component.
func (c Component) KubeConfigPath() string {
	return fmt.Sprintf("/var/lib/%s/kubeconfig", c)
}

// JobID returns the ID of the vault-monkey job used to access certificates for this component.
func (c Component) JobID(clusterID string) string {
	return jobID(clusterID, c.Name())
}

// RestartCommand returns a full command that restarts the component
func (c Component) RestartCommand() string {
	if c.isManifest {
		return fmt.Sprintf("/usr/bin/touch %s", c.ManifestPath())
	} else {
		return fmt.Sprintf("/bin/systemctl restart %s", c.ServiceName())
	}
}
