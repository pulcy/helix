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

package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
)

var (
	maskAny = errors.WithStack
)

const (
	waitTimeout = time.Minute * 10
)

func NewService() service.Service {
	return &cpService{}
}

type cpService struct {
}

func (t *cpService) Name() string {
	return "control-plane"
}

func (t *cpService) Prepare(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

// Init waits for the control plane to become responsive.
func (t *cpService) Init(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger

	client, err := service.NewKubernetesClient(sctx, deps, flags)
	if err != nil {
		return maskAny(err)
	}
	log.Info().Msg("Waiting for control-plane to respond")
	start := time.Now()
	for {
		if time.Since(start) > waitTimeout {
			return maskAny(fmt.Errorf("Control-plane did not wake up in time"))
		}
		op := func() error {
			var nodes corev1.NodeList
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			if err := client.List(ctx, k8s.AllNamespaces, &nodes); err != nil {
				return maskAny(err)
			}
			// Got a good response
			if len(nodes.Items) == 0 {
				return maskAny(fmt.Errorf("No nodes yet"))
			}
			return nil
		}
		if err := op(); err != nil {
			time.Sleep(time.Second)
		} else {
			// Got good response
			break
		}
	}
	log.Info().Msg("Control-plane is responding")

	return nil
}
