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

package proxy

import (
	"context"
	"fmt"

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	rbacv1 "github.com/ericchiang/k8s/apis/rbac/v1"

	"github.com/pkg/errors"

	"github.com/pulcy/helix/service"
	"github.com/pulcy/helix/util"
)

var (
	maskAny = errors.WithStack
)

func NewService() service.Service {
	return &proxyService{}
}

type proxyService struct {
}

func (t *proxyService) Name() string {
	return "kube-proxy"
}

func (t *proxyService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

func (t *proxyService) Init(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	ctx := context.Background()
	log := deps.Logger
	client, err := service.NewKubernetesClient(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create kube-proxy service account
	sa := &corev1.ServiceAccount{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("kube-proxy"),
			Namespace: k8s.String("kube-system"),
		},
	}
	if err := util.CreateOrUpdate(ctx, client, sa); err != nil {
		return maskAny(err)
	}

	// Create kube-proxy cluster role binding
	crb := &rbacv1.ClusterRoleBinding{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String("kube-proxy"),
		},
		RoleRef: &rbacv1.RoleRef{
			ApiGroup: k8s.String("rbac.authorization.k8s.io"),
			Kind:     k8s.String("ClusterRole"),
			Name:     k8s.String("system:node-proxier"), // Automatically created system role.
		},
		Subjects: []*rbacv1.Subject{
			&rbacv1.Subject{
				Kind:      k8s.String("ServiceAccount"),
				Name:      k8s.String("kube-proxy"),
				Namespace: k8s.String("kube-system"),
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, crb); err != nil {
		return maskAny(err)
	}

	// Render kubeconfig to kube-proxy
	kubeConfigOpts := struct {
		MasterEndpoint string
	}{
		MasterEndpoint: fmt.Sprintf("https://%s:6443", flags.ControlPlane.GetAPIServerAddress()),
	}
	kubeconfig, err := util.RenderToString(log, kubeConfigTemplate, kubeConfigOpts)
	if err != nil {
		return maskAny(err)
	}

	// Create kube-proxy config map
	cm := &corev1.ConfigMap{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("kube-proxy"),
			Namespace: k8s.String("kube-system"),
			Labels: map[string]string{
				"app": "kube-proxy",
			},
		},
		Data: map[string]string{
			"kubeconfig.conf": kubeconfig,
		},
	}
	if err := util.CreateOrUpdate(ctx, client, cm); err != nil {
		return maskAny(err)
	}

	// Create kube-proxy daemon-set
	for _, arch := range flags.AllArchitectures() {
		ds := &appsv1.DaemonSet{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("kube-proxy-" + arch),
				Namespace: k8s.String("kube-system"),
				Labels: map[string]string{
					"k8s-app":                 "kube-proxy",
					"beta.kubernetes.io/arch": arch,
				},
			},
			Spec: &appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"k8s-app":                 "kube-proxy",
						"beta.kubernetes.io/arch": arch,
					},
				},
				UpdateStrategy: &appsv1.DaemonSetUpdateStrategy{
					Type: k8s.String("RollingUpdate"),
				},
				Template: &corev1.PodTemplateSpec{
					Metadata: &metav1.ObjectMeta{
						Labels: map[string]string{
							"k8s-app":                 "kube-proxy",
							"beta.kubernetes.io/arch": arch,
						},
					},
					Spec: &corev1.PodSpec{
						NodeSelector: map[string]string{
							"beta.kubernetes.io/arch": arch,
						},
						Containers: []*corev1.Container{
							&corev1.Container{
								Name:  k8s.String("kube-proxy"),
								Image: k8s.String(flags.Images.HyperKubeImage(arch)),
								Command: []string{
									"/hyperkube",
									"kube-proxy",
									"--kubeconfig=/var/lib/kube-proxy/kubeconfig.conf",
									"--proxy-mode=iptables",
								}, // TODO
								SecurityContext: &corev1.SecurityContext{
									Privileged: k8s.Bool(true),
								},
								VolumeMounts: []*corev1.VolumeMount{
									&corev1.VolumeMount{
										MountPath: k8s.String("/var/lib/kube-proxy"),
										Name:      k8s.String("kube-proxy"),
									},
									&corev1.VolumeMount{
										MountPath: k8s.String("/run/xtables.lock"),
										Name:      k8s.String("xtables-lock"),
										ReadOnly:  k8s.Bool(true),
									},
									&corev1.VolumeMount{
										MountPath: k8s.String("/lib/modules"),
										Name:      k8s.String("lib-modules"),
									},
								},
							},
						},
						HostNetwork:        k8s.Bool(true),
						ServiceAccountName: k8s.String("kube-proxy"),
						Tolerations:        []*corev1.Toleration{
						// TODO
						},
						Volumes: []*corev1.Volume{
							&corev1.Volume{
								Name: k8s.String("kube-proxy"),
								VolumeSource: &corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: &corev1.LocalObjectReference{
											Name: k8s.String("kube-proxy"),
										},
									},
								},
							},
							&corev1.Volume{
								Name: k8s.String("xtables-lock"),
								VolumeSource: &corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: k8s.String("/run/xtables.lock"),
										Type: k8s.String("FileOrCreate"),
									},
								},
							},
							&corev1.Volume{
								Name: k8s.String("lib-modules"),
								VolumeSource: &corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: k8s.String("/lib/modules"),
									},
								},
							},
						},
					},
				},
			},
		}
		if err := util.CreateOrUpdate(ctx, client, ds); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
