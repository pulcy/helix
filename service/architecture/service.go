// Copyright (c) 2016 Pulcy.
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

package architecture

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

func NewService() service.Service {
	return &archService{}
}

type archService struct {
}

func (t *archService) Name() string {
	return "architecture"
}

func (t *archService) Prepare(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

// InitNode detects the architecture of the node.
func (t *archService) InitNode(node *service.Node, client util.SSHClient, sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	log := deps.Logger.With().Str("host", node.Name).Logger()

	if node.Architecture == "" {
		result, err := client.Run(log, "uname -p", "", true)
		if err != nil {
			return maskAny(err)
		}
		result = strings.TrimSpace(result)
		switch result {
		case "armv7l":
			node.Architecture = "arm"
		case "x86_64":
			node.Architecture = "amd64"
		default:
			return maskAny(fmt.Errorf("Unsupported architecture '%s'", result))
		}
		log.Info().Msgf("Found '%s' architecture on node", node.Architecture)
	}

	return nil
}
