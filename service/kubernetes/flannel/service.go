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

package flannel

import (
	"context"

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
	return &flannelService{}
}

type flannelService struct {
}

func (t *flannelService) Name() string {
	return "flannel"
}

func (t *flannelService) Prepare(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

func (t *flannelService) Init(sctx *service.ServiceContext, deps service.ServiceDependencies, flags service.ServiceFlags) error {
	ctx := context.Background()
	log := deps.Logger
	client, err := service.NewKubernetesClient(sctx, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create flannel service account
	sa := &corev1.ServiceAccount{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("flannel"),
			Namespace: k8s.String("kube-system"),
		},
	}
	if err := util.CreateOrUpdate(ctx, client, sa); err != nil {
		return maskAny(err)
	}

	// Create flannel cluster role
	cr := &rbacv1.ClusterRole{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String("flannel"),
		},
		Rules: []*rbacv1.PolicyRule{
			&rbacv1.PolicyRule{
				ApiGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
			&rbacv1.PolicyRule{
				ApiGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"list", "watch"},
			},
			&rbacv1.PolicyRule{
				ApiGroups: []string{""},
				Resources: []string{"nodes/status"},
				Verbs:     []string{"patch"},
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, cr); err != nil {
		return maskAny(err)
	}

	// Create flannel cluster role binding
	crb := &rbacv1.ClusterRoleBinding{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String("flannel"),
		},
		RoleRef: &rbacv1.RoleRef{
			ApiGroup: k8s.String("rbac.authorization.k8s.io"),
			Kind:     k8s.String("ClusterRole"),
			Name:     k8s.String("flannel"),
		},
		Subjects: []*rbacv1.Subject{
			&rbacv1.Subject{
				Kind:      k8s.String("ServiceAccount"),
				Name:      k8s.String("flannel"),
				Namespace: k8s.String("kube-system"),
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, crb); err != nil {
		return maskAny(err)
	}

	// Render cni-conf.json
	cniConfOpts := struct{}{}
	cniConf, err := util.RenderToString(log, cniConfTemplate, cniConfOpts)
	if err != nil {
		return maskAny(err)
	}

	// Render net-conf.json
	netConfOpts := struct{}{}
	netConf, err := util.RenderToString(log, netConfTemplate, netConfOpts)
	if err != nil {
		return maskAny(err)
	}

	// Create flannel config map
	cm := &corev1.ConfigMap{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("flannel"),
			Namespace: k8s.String("kube-system"),
			Labels: map[string]string{
				"tier": "node",
				"app":  "flannel",
			},
		},
		Data: map[string]string{
			"cni-conf.json": cniConf,
			"net-conf.json": netConf,
		},
	}
	if err := util.CreateOrUpdate(ctx, client, cm); err != nil {
		return maskAny(err)
	}

	// Create flannel daemonset
	for _, arch := range sctx.AllArchitectures() {
		ds := &appsv1.DaemonSet{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("kube-flannel-ds-" + arch),
				Namespace: k8s.String("kube-system"),
				Labels: map[string]string{
					"tier": "node",
					"app":  "flannel",
					"beta.kubernetes.io/arch": arch,
				},
			},
			Spec: &appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"tier": "node",
						"app":  "flannel",
						"beta.kubernetes.io/arch": arch,
					},
				},
				Template: &corev1.PodTemplateSpec{
					Metadata: &metav1.ObjectMeta{
						Labels: map[string]string{
							"tier": "node",
							"app":  "flannel",
							"beta.kubernetes.io/arch": arch,
						},
					},
					Spec: &corev1.PodSpec{
						HostNetwork: k8s.Bool(true),
						NodeSelector: map[string]string{
							"beta.kubernetes.io/arch": arch,
						},
						Tolerations: []*corev1.Toleration{
							&corev1.Toleration{
								Key:      k8s.String("node-role.kubernetes.io/master"),
								Operator: k8s.String("Exists"),
								Effect:   k8s.String("NoSchedule"),
							},
						},
						ServiceAccountName: k8s.String("flannel"),
						InitContainers: []*corev1.Container{
							&corev1.Container{
								Name:    k8s.String("install-cni"),
								Image:   k8s.String(flags.Images.FlannelImage(arch)),
								Command: []string{"cp"},
								Args: []string{
									"-f",
									"/etc/kube-flannel/cni-conf.json",
									"/etc/cni/net.d/10-flannel.conf",
								},
								VolumeMounts: []*corev1.VolumeMount{
									&corev1.VolumeMount{
										MountPath: k8s.String("/etc/cni/net.d"),
										Name:      k8s.String("cni"),
									},
									&corev1.VolumeMount{
										MountPath: k8s.String("/etc/kube-flannel/"),
										Name:      k8s.String("flannel-cfg"),
									},
								},
							},
						},
						Containers: []*corev1.Container{
							&corev1.Container{
								Name:  k8s.String("kube-flannel"),
								Image: k8s.String(flags.Images.FlannelImage(arch)),
								Command: []string{
									"/opt/bin/flanneld",
									"--ip-masq",
									"--kube-subnet-mgr",
								},
								Env: []*corev1.EnvVar{
									&corev1.EnvVar{
										Name:      k8s.String("POD_NAME"),
										ValueFrom: util.EnvVarSourceFieldRef("metadata.name"),
									},
									&corev1.EnvVar{
										Name:      k8s.String("POD_NAMESPACE"),
										ValueFrom: util.EnvVarSourceFieldRef("metadata.namespace"),
									},
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: k8s.Bool(true),
								},
								VolumeMounts: []*corev1.VolumeMount{
									&corev1.VolumeMount{
										MountPath: k8s.String("/run"),
										Name:      k8s.String("run"),
									},
									&corev1.VolumeMount{
										MountPath: k8s.String("/etc/kube-flannel/"),
										Name:      k8s.String("flannel-cfg"),
									},
								},
							},
						},
						Volumes: []*corev1.Volume{
							&corev1.Volume{
								Name: k8s.String("run"),
								VolumeSource: &corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: k8s.String("/run"),
									},
								},
							},
							&corev1.Volume{
								Name: k8s.String("cni"),
								VolumeSource: &corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: k8s.String("/etc/cni/net.d"),
									},
								},
							},
							&corev1.Volume{
								Name: k8s.String("flannel-cfg"),
								VolumeSource: &corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: &corev1.LocalObjectReference{
											Name: k8s.String("flannel"),
										},
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
