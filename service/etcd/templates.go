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

package etcd

const (
	etcdManifestTemplate = `apiVersion: v1
kind: Pod
metadata:
labels:
	component: etcd
	tier: control-plane
name: {{.PodName}}
namespace: kube-system
spec:
containers:
- command:
	- etcd --name {{.PeerName}} \
	- --data-dir {{.DataDir}} \
	- --listen-client-urls https://${PRIVATE_IP}:2379 \
	- --advertise-client-urls https://${PRIVATE_IP}:2379 \
	- --listen-peer-urls https://${PRIVATE_IP}:2380 \
	- --initial-advertise-peer-urls https://${PRIVATE_IP}:2380 \
	- --cert-file={{.ClientCertFile}} \
	- --key-file={{.ClientKeyFile}} \
	- --client-cert-auth \
	- --trusted-ca-file={{.ClientCAFile}} \
	- --peer-cert-file={{.PeerCertFile}} \
	- --peer-key-file={{.PeerKeyFile}} \
	- --peer-client-cert-auth \
	- --peer-trusted-ca-file={{.PeerCAFile}} \
	- --initial-cluster {{.InitialCluster}} \
	- --initial-cluster-token {{.InitialClusterToken}} \
	- --initial-cluster-state {{.ClusterState}}
	image: {{.Image}}
	livenessProbe:
	httpGet:
		path: /health
		port: 2379
		scheme: HTTP
	initialDelaySeconds: 15
	timeoutSeconds: 15
	name: etcd
	env:
	- name: PUBLIC_IP
	valueFrom:
		fieldRef:
		fieldPath: status.hostIP
	- name: PRIVATE_IP
	valueFrom:
		fieldRef:
		fieldPath: status.podIP
	volumeMounts:
	- mountPath: {{.DataDir}}
	name: etcd
	- mountPath: {{.CertificatesDir}}
	name: certs
hostNetwork: true
volumes:
- hostPath:
	path: {{.DataDir}}
	type: DirectoryOrCreate
	name: etcd
- hostPath:
	path: {{.CertificatesDir}}
	name: certs
`
)
