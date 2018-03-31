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

package apiserver

const (
	apiserverManifestTemplate = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    component: kube-apiserver
    tier: control-plane
  name: {{.PodName}}
  namespace: kube-system
spec:
  containers:
  - command:
    - /hyperkube
    - kube-apiserver
    - --tls-cert-file={{ .CertFile }}
    - --tls-private-key-file={{ .KeyFile }}
    - --secure-port=6443
    - --insecure-port=0
    - --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
    - --client-ca-file={{ .ClientCAFile }}
    - --requestheader-client-ca-file={{.ProxyClientCAFile}}
    - --proxy-client-cert-file={{.ProxyClientCertFile}}
    - --proxy-client-key-file={{.ProxyClientKeyFile}}
    - --requestheader-extra-headers-prefix=X-Remote-Extra-
    - --requestheader-allowed-names=front-proxy-client
    - --service-cluster-ip-range=10.96.0.0/12
    - --kubelet-certificate-authority={{ .KubeletCAFile }}
    - --kubelet-client-certificate={{ .KubeletCertFile }}
    - --kubelet-client-key={{ .KubeletKeyFile }}
    - --requestheader-username-headers=X-Remote-User
    - --advertise-address=$(PUBLIC_IP)
    - --admission-control=Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,ResourceQuota
    - --enable-bootstrap-token-auth=true
    - --allow-privileged=true
    - --requestheader-group-headers=X-Remote-Group
    - --authorization-mode=Node,RBAC
    - --etcd-cafile={{ .EtcdCAFile }}
    - --etcd-certfile={{ .EtcdCertFile }}
    - --etcd-keyfile={{ .EtcdKeyFile }}
    - --etcd-servers={{ .EtcdEndpoints }}
    - --feature-gates={{ .FeatureGates }}
    image: {{ .Image }}
    env:
    - name: PUBLIC_IP
      valueFrom: 
        fieldRef:
          fieldPath: status.hostIP
    - name: PRIVATE_IP
      valueFrom: 
        fieldRef:
          fieldPath: status.podIP
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: $(PUBLIC_IP)
        path: /healthz
        port: 6443
        scheme: HTTPS
      initialDelaySeconds: 15
      timeoutSeconds: 15
    name: kube-apiserver
    resources:
      requests:
        cpu: 250m
    volumeMounts:
    - mountPath: {{ .PkiDir }}
      name: k8s-certs
      readOnly: true
    - mountPath: /etc/ssl/certs
      name: ca-certs
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: {{ .PkiDir }}
      type: DirectoryOrCreate
    name: k8s-certs
  - hostPath:
      path: /etc/ssl/certs
      type: DirectoryOrCreate
    name: ca-certs
`
)
