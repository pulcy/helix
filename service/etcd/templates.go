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
	- etcd --name ${PEER_NAME} \
	- --data-dir /var/lib/etcd \
	- --listen-client-urls https://${PRIVATE_IP}:2379 \
	- --advertise-client-urls https://${PRIVATE_IP}:2379 \
	- --listen-peer-urls https://${PRIVATE_IP}:2380 \
	- --initial-advertise-peer-urls https://${PRIVATE_IP}:2380 \
	- --cert-file=/certs/server.pem \
	- --key-file=/certs/server-key.pem \
	- --client-cert-auth \
	- --trusted-ca-file=/certs/ca.pem \
	- --peer-cert-file=/certs/peer.pem \
	- --peer-key-file=/certs/peer-key.pem \
	- --peer-client-cert-auth \
	- --peer-trusted-ca-file=/certs/ca.pem \
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
	- name: PEER_NAME
	valueFrom:
		fieldRef:
		fieldPath: metadata.name
	volumeMounts:
	- mountPath: /var/lib/etcd
	name: etcd
	- mountPath: /certs
	name: certs
hostNetwork: true
volumes:
- hostPath:
	path: /var/lib/etcd
	type: DirectoryOrCreate
	name: etcd
- hostPath:
	path: /etc/kubernetes/pki/etcd
	name: certs
`
)
