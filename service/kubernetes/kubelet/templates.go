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

package kubelet

const (
	kubeletServiceTemplate = `[Unit]
Description=Kubernetes Kubelet Server
Requires=docker.service network-online.target
After=hyperkube.service docker.service network-online.target

[Service]
ExecStartPre=/bin/mkdir -p /opt/log/pods
ExecStartPre=/bin/mount --bind /var/log/pods /opt/log/pods
ExecStartPre=/bin/mkdir -p /opt/log/containers
ExecStartPre=/bin/mount --bind /var/log/containers /opt/log/containers
ExecStartPre=/bin/mkdir -p /var/lib/kubelet
#ExecStartPre=/bin/mount --bind /var/lib/kubelet /var/lib/kubelet
#ExecStartPre=/bin/mount --make-shared /var/lib/kubelet
ExecStart=/usr/local/bin/hyperkube-{{ .KubernetesVersion }} kubelet \
		--allow-privileged=true \
		--authorization-mode=AlwaysAllow \
		--bootstrap-kubeconfig={{.BootstrapKubeConfigPath}} \
		--cgroup-driver=cgroupfs \
		--client-ca-file={{.ClientCAPath}} \
		--cloud-provider= \
		--cluster-dns={{.ClusterDNS}} \
		--cluster-domain={{.ClusterDomain}} \
		--cni-bin-dir=/opt/cni/bin \
		--cni-conf-dir=/etc/cni/net.d \
		--container-runtime=docker \
		--feature-gates={{.FeatureGates}} \
		--hairpin-mode=none \
		--kubeconfig={{.KubeConfigPath}} \
		--network-plugin=cni \
		--node-labels={{.NodeLabels}} \
		--pod-manifest-path=/etc/kubernetes/manifests \
		--register-node=true \
		--rotate-certificates=true \
		--tls-cert-file={{.CertPath}} \
		--tls-private-key-file={{.KeyPath}} \
		--v=2	  
Restart=always
StartLimitInterval=0
RestartSec=10
KillMode=process

[Install]
WantedBy=multi-user.target`
)
