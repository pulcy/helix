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
	kubeLogrotateServiceTemplate = "templates/kubernetes/kube-logrotate.service.tmpl"
	kubeLogrotateTimerTemplate   = "templates/kubernetes/kube-logrotate.timer.tmpl"
	kubeLogrotateConfTemplate    = "templates/kubernetes/kube-logrotate.conf.tmpl"
	kubeLogrotateConfPath        = "/etc/logrotate.d/kube-logrotate.conf"
)

// createKubeLogrotateService creates the file containing the kubernetes Kube-logrotate service.
func createKubeLogrotateService(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	deps.Logger.Info("creating %s", kubeLogrotateConfPath)
	confChanged, err := templates.Render(deps.Logger, kubeLogrotateConfTemplate, kubeLogrotateConfPath, nil, configFileMode)

	deps.Logger.Info("creating %s", c.ServicePath())
	serviceChanged, err := templates.Render(deps.Logger, kubeLogrotateServiceTemplate, c.ServicePath(), nil, serviceFileMode)

	deps.Logger.Info("creating %s", c.TimerPath())
	timerChanged, err := templates.Render(deps.Logger, kubeLogrotateTimerTemplate, c.TimerPath(), nil, serviceFileMode)
	return confChanged || serviceChanged || timerChanged, maskAny(err)
}
