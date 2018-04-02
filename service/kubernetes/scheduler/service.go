// Copyright (c) 2018 Pulcy.
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

package scheduler

import (
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/service/kubernetes/component"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

const (
	manifestPath = "/etc/kubernetes/manifests/kube-scheduler.yaml"

	manifestFileMode = os.FileMode(0644)
	certFileMode     = util.CertFileMode
	keyFileMode      = util.KeyFileMode
)

func NewService() service.Service {
	return &schedulerService{}
}

type schedulerService struct {
	component.Component
}

func (t *schedulerService) Name() string {
	return "kube-scheduler"
}

func (t *schedulerService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	t.Component.Name = "scheduler"
	return nil
}

// InitMachine configures the machine to run kube-scheduler.
func (t *schedulerService) InitMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	// Setup scheduler on this host?
	if !node.IsControlPlane {
		log.Info().Msg("No kube-scheduler on this machine")
		return nil
	}

	cfg, err := t.createConfig(node, client, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create & Upload kubeconfig
	if err := t.Component.CreateKubeConfig("system:kube-scheduler", "Kubernetes", client, deps, flags); err != nil {
		return maskAny(err)
	}

	// Create manifest
	log.Info().Msg("Creating kube-apiserver manifest")
	if err := createManifest(client, deps, cfg); err != nil {
		return maskAny(err)
	}

	return nil
}

// ResetMachine removes kube-scheduler from the machine.
func (t *schedulerService) ResetMachine(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	if err := client.RemoveFile(deps.Logger, manifestPath); err != nil {
		return maskAny(err)
	}
	if err := t.Component.RemoveKubeConfig(client, deps, flags); err != nil {
		return maskAny(err)
	}
	return nil
}

type config struct {
	Image          string // HyperKube docker images
	PodName        string
	FeatureGates   string // Feature gates to use
	KubeConfigPath string // Path to a kubeconfig file, specifying how to connect to the API server.
}

func (t *schedulerService) createConfig(node service.Node, client util.SSHClient, deps service.ServiceDependencies, flags service.ServiceFlags) (config, error) {
	result := config{
		Image:          flags.Images.HyperKube,
		PodName:        "kube-scheduler-" + node.Name,
		FeatureGates:   strings.Join(flags.Kubernetes.FeatureGates, ","),
		KubeConfigPath: t.KubeConfigPath(),
	}

	return result, nil
}

func createManifest(client util.SSHClient, deps service.ServiceDependencies, opts config) error {
	deps.Logger.Info().Msgf("Creating manifest %s", manifestPath)
	if err := client.Render(deps.Logger, schedulerManifestTemplate, manifestPath, opts, manifestFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
