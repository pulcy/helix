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
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
)

const (
	compNameKubelet               = "kubelet"
	compNameKubeProxy             = "kube-proxy"
	compNameKubeServiceAccounts   = "kube-serviceaccounts"
	compNameKubeAPIServer         = "kube-apiserver"
	compNameKubeControllerManager = "kube-controller-manager"
	compNameKubeScheduler         = "kube-scheduler"
	compNameKubeAddonManager      = "kube-addon-manager"
	compNameKubeDNS               = "kube-dns"
	compNameKubeLogrotate         = "kube-logrotate"
)

var (
	maskAny = errors.WithStack

	components = map[Component]componentSetup{
		// Components that should be installed on all nodes
		NewServiceComponent(compNameKubelet, false):               componentSetup{createKubeletService, nil, false, true},
		NewServiceComponent(compNameKubeProxy, false):             componentSetup{createKubeProxyService, nil, false, true},
		NewServiceAndTimerComponent(compNameKubeLogrotate, false): componentSetup{createKubeLogrotateService, nil, false, false},
		// Components that should be installed on master nodes only
		NewManifestComponent(compNameKubeAPIServer, true):         componentSetup{createKubeApiServerManifest, createKubeApiServerAltNames, true, true},
		NewManifestComponent(compNameKubeControllerManager, true): componentSetup{createKubeControllerManagerManifest, nil, false, true},
		NewManifestComponent(compNameKubeScheduler, true):         componentSetup{createKubeSchedulerManifest, nil, false, true},
		NewManifestComponent(compNameKubeAddonManager, true):      componentSetup{createKubeAddonManagerManifest, nil, false, false},
		NewManifestComponent(compNameKubeDNS, true):               componentSetup{createKubeDNSAddon, nil, false, false},
	}
)

const (
	configFileMode   = os.FileMode(0644)
	manifestFileMode = os.FileMode(0644)
	serviceFileMode  = os.FileMode(0644)
	templateFileMode = os.FileMode(0400)
)

type componentSetup struct {
	Setup                  func(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error)
	GetAltNames            func(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) []string
	AddInternalApiServerIP bool
	CreateCertificates     bool
}

func NewService() service.Service {
	return &k8sService{}
}

type k8sService struct{}

func (t *k8sService) Name() string {
	return "kubernetes"
}

func (t *k8sService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	runKubernetes := flags.Kubernetes.IsEnabled()
	if runKubernetes {
		// Ensure copied CNI binaries are linked to /opt/cni/bin
		if err := linkCniBinaries(deps, flags); err != nil {
			return maskAny(err)
		}

		// Install template & service that extracts the service-accounts-key-file
		serviceAccountsTemplateChanged, err := createServiceAccountsTemplate(deps, flags)
		if err != nil {
			return maskAny(err)
		}
		serviceAccountsServiceChanged, err := createServiceAccountsService(deps, flags)
		if err != nil {
			return maskAny(err)
		}
		isActive, err := deps.Systemd.IsActive(serviceAccountsTokenServiceName)
		if err != nil {
			return maskAny(err)
		}

		if !isActive || serviceAccountsTemplateChanged || serviceAccountsServiceChanged || flags.Force {
			if err := deps.Systemd.Enable(serviceAccountsTokenServiceName); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Reload(); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Restart(serviceAccountsTokenServiceName); err != nil {
				return maskAny(err)
			}
		}
	}
	for c, compSetup := range components {
		installComponent := runKubernetes
		if c.MasterOnly() && !flags.HasRole("core") {
			installComponent = false
		}
		var certsTimerChanged, certsServiceChanged bool
		if installComponent {
			if compSetup.CreateCertificates {
				// Create k8s-*-certs.service and template file
				var err error
				var altNames []string
				if compSetup.GetAltNames != nil {
					altNames = compSetup.GetAltNames(deps, flags, c)
				}
				if certsServiceChanged, err = createCertsService(deps, flags, c, altNames, compSetup.AddInternalApiServerIP); err != nil {
					return maskAny(err)
				}
				if certsTimerChanged, err = createCertsTimer(deps, flags, c); err != nil {
					return maskAny(err)
				}
				isActive, err := deps.Systemd.IsActive(c.CertificatesServiceName())
				if err != nil {
					return maskAny(err)
				}

				if !isActive || certsServiceChanged || certsTimerChanged || flags.Force {
					if err := deps.Systemd.Enable(c.CertificatesServiceName()); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Enable(c.CertificatesTimerName()); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Reload(); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Restart(c.CertificatesServiceName()); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Restart(c.CertificatesTimerName()); err != nil {
						return maskAny(err)
					}
				}
			}

			// Create component service / manifest
			if compSetup.Setup != nil {
				serviceChanged, err := compSetup.Setup(deps, flags, c)
				if err != nil {
					return maskAny(err)
				}

				if !c.IsManifest() {
					isActive, err := deps.Systemd.IsActive(c.ServiceName())
					if err != nil {
						return maskAny(err)
					}

					if !isActive || serviceChanged || certsServiceChanged || flags.Force {
						if err := deps.Systemd.Enable(c.ServiceName()); err != nil {
							return maskAny(err)
						}
						if c.HasTimer() {
							if err := deps.Systemd.Enable(c.TimerName()); err != nil {
								return maskAny(err)
							}
						}
						if err := deps.Systemd.Reload(); err != nil {
							return maskAny(err)
						}
						if err := deps.Systemd.Restart(c.ServiceName()); err != nil {
							return maskAny(err)
						}
						if c.HasTimer() {
							if err := deps.Systemd.Restart(c.TimerName()); err != nil {
								return maskAny(err)
							}
						}
					}
				}
			}
		} else {
			// Component service no longer needed, remove it
			if c.IsManifest() {
				os.Remove(c.ManifestPath())
			} else {
				if c.HasTimer() {
					if exists, err := deps.Systemd.Exists(c.TimerName()); err != nil {
						return maskAny(err)
					} else if exists {
						if err := deps.Systemd.Disable(c.TimerName()); err != nil {
							deps.Logger.Errorf("Disabling %s failed: %#v", c.TimerName(), err)
						} else {
							os.Remove(c.TimerPath())
						}
					}
				}
				if exists, err := deps.Systemd.Exists(c.ServiceName()); err != nil {
					return maskAny(err)
				} else if exists {
					if err := deps.Systemd.Disable(c.ServiceName()); err != nil {
						deps.Logger.Errorf("Disabling %s failed: %#v", c.ServiceName(), err)
					} else {
						os.Remove(c.ServicePath())
					}
				}
			}

			// k8s-*-certs.timer no longer needed, remove it
			if exists, err := deps.Systemd.Exists(c.CertificatesTimerName()); err != nil {
				return maskAny(err)
			} else if exists {
				if err := deps.Systemd.Disable(c.CertificatesTimerName()); err != nil {
					deps.Logger.Errorf("Disabling %s failed: %#v", c.CertificatesTimerName(), err)
				} else {
					os.Remove(c.CertificatesTimerPath())
				}
			}

			// k8s-*-certs.service no longer needed, remove it
			if exists, err := deps.Systemd.Exists(c.CertificatesServiceName()); err != nil {
				return maskAny(err)
			} else if exists {
				if err := deps.Systemd.Disable(c.CertificatesServiceName()); err != nil {
					deps.Logger.Errorf("Disabling %s failed: %#v", c.CertificatesServiceName(), err)
				} else {
					os.Remove(c.CertificatesServicePath())
				}
			}
		}
	}

	return nil
}

// getAPIServers creates a list of URL to the API servers of the cluster.
func getAPIServers(deps service.ServiceDependencies, flags *service.ServiceFlags) ([]string, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return nil, maskAny(err)
	}
	var apiServers []string
	for _, m := range members {
		if !m.EtcdProxy {
			apiServers = append(apiServers, fmt.Sprintf("https://%s:%d", m.ClusterIP, flags.Kubernetes.APIServerPort))
		}
	}
	return apiServers, nil
}

// servicePath returns the full path of the file containing the service with given name.
func servicePath(serviceName string) string {
	return "/etc/systemd/system/" + serviceName
}

// manifestPath returns the full path of the file containing the manifest with given name.
func manifestPath(manifestName string) string {
	return "/etc/kubernetes/manifests/" + manifestName
}

// addonPath returns the full path of the file containing the addon with given name.
func addonPath(addonName string) string {
	return "/etc/kubernetes/addons/" + addonName
}

// certificatePath returns the full path of the file with given name.
func certificatePath(fileName string) string {
	return "/opt/certs/" + fileName
}

// jobID returns the ID of the vault-monkey job used to access certificates for the given component.
func jobID(clusterID, componentName string) string {
	return fmt.Sprintf("ca-%s-pki-k8s-%s", clusterID, componentName)
}

// tokenRole returns the name of the role used to extract a token by vault-monkey.
func tokenRole(clusterID, componentName string) string {
	return fmt.Sprintf("k8s-%s-%s", clusterID, componentName)
}

// tokenPolicy returns the name of the policy used to extract a token by vault-monkey.
func tokenPolicy(clusterID, componentName string) string {
	return path.Join("ca", clusterID, "pki/k8s", componentName)
}
