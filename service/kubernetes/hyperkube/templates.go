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

package hyperkube

const (
	hyperkubeServiceTemplate = `[Unit]
Description=Kubernetes Hyperkube Installer
Requires=docker.service network-online.target
After=docker.service network-online.target

[Service]
Type=oneshot
ExecStartPre=/bin/mkdir -p /usr/local/bin
ExecStartPre=/bin/sh -c "test -f {{ .HyperKubePath }} || /usr/bin/docker run --rm -v /usr/local/bin:/usr/local/bin {{.Image}} cp /hyperkube {{ .HyperKubePath }}"
ExecStart=/bin/sh -c "test -e {{ .KubeCtlPath }} || ln -sf {{ .HyperKubePath }} {{ .KubeCtlPath }}"
Restart=no
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target`
)
