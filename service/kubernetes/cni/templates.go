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

package cni

const (
	cniDownloadServiceTemplate = `[Unit]
Description=CNI Installer
Requires=docker.service network-online.target
After=docker.service network-online.target

[Service]
Type=oneshot
ExecStartPre=/bin/mkdir -p {{ .CniBinDir }}
ExecStartPre=/bin/sh -c "test -f {{ .PluginsTgzPath }} || wget -O {{ .PluginsTgzPath }} {{ .PluginsURL }}"
ExecStart=/bin/sh -c "test -e {{ .CniBinDir }}/loopback || tar -xvf {{ .PluginsTgzPath }} -C {{ .CniBinDir }}/"
Restart=no
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target`
)
