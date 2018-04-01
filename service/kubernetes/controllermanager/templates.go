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

package controllermanager

const (
	controllermanagerManifestTemplate = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    component: kube-controller-manager
    tier: control-plane
  name: {{ .PodName }}
  namespace: kube-system
spec:
  containers:
  - command:
    - /hyperkube
    - kube-controller-manager
    - --address=127.0.0.1
    - --cluster-signing-key-file={{ .ClusterSigningKeyFile }}
    - --cluster-signing-cert-file={{ .ClusterSigningCertFile }}
    - --leader-elect=true
    - --use-service-account-credentials=true
    - --controllers=*,bootstrapsigner,tokencleaner
    - --kubeconfig={{ .KubeConfigPath }}
    - --root-ca-file={{ .RootCAFile }}
    - --service-account-private-key-file={{ .ServiceAccountKeyFile }}
    - --allocate-node-cidrs=true
    - --cluster-cidr=10.244.0.0/16
    - --node-cidr-mask-size=24
    - --feature-gates={{ .FeatureGates }}
    image: {{ .Image }}
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10252
        scheme: HTTP
      initialDelaySeconds: 15
      timeoutSeconds: 15
    name: kube-controller-manager
    resources:
      requests:
        cpu: 200m
    volumeMounts:
    - mountPath: {{ .PkiDir }}
      name: k8s-certs
      readOnly: true
    - mountPath: /etc/ssl/certs
      name: ca-certs
      readOnly: true
    - mountPath: {{ .KubeConfigPath }}
      name: kubeconfig
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
  - hostPath:
      path: {{ .KubeConfigPath }}
      type: FileOrCreate
    name: kubeconfig
`
)
