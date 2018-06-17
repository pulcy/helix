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

package keepalived

const (
	keepalivedConfTemplate = `
! Configuration File for keepalived
global_defs {
  router_id LVS_DEVEL
}

vrrp_script check_apiserver {
  script "/etc/keepalived/check_apiserver.sh"
  interval 3
  weight -2
  fall 10
  rise 2
}

vrrp_instance VI_1 {
  state {{.State}}
  interface {{.Interface}}
  virtual_router_id 51
  priority {{.Priority}}
  authentication {
      auth_type PASS
      auth_pass {{.AuthPassword}}
  }
  virtual_ipaddress {
    {{.VirtualIP}}
  }
  track_script {
      check_apiserver
  }
}
`

	checkAPIServerTemplate = `#!/bin/sh

errorExit() {
  echo "*** $*" 1>&2
  exit 1
}

curl --silent --max-time 2 --insecure https://localhost:6443/ -o /dev/null || errorExit "Error GET https://localhost:6443/"
if ip addr | grep -q {{.VirtualIP}}; then
  curl --silent --max-time 2 --insecure https://{{.VirtualIP}}:6443/ -o /dev/null || errorExit "Error GET https://{{.VirtualIP}}:6443/"
fi
`
)
