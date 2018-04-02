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

package coredns

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
	return &dnsService{}
}

type dnsService struct {
}

func (t *dnsService) Name() string {
	return "coredns"
}

func (t *dnsService) Prepare(deps service.ServiceDependencies, flags service.ServiceFlags, willInit bool) error {
	return nil
}

func (t *dnsService) Init(deps service.ServiceDependencies, flags service.ServiceFlags) error {
	ctx := context.Background()
	log := deps.Logger
	client, err := service.NewKubernetesClient(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	// Create coredns service account
	sa := &corev1.ServiceAccount{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("coredns"),
			Namespace: k8s.String("kube-system"),
		},
	}
	if err := util.CreateOrUpdate(ctx, client, sa); err != nil {
		return maskAny(err)
	}

	// Create system:coredns cluster role
	cr := &rbacv1.ClusterRole{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String("system:coredns"),
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
		},
		Rules: []*rbacv1.PolicyRule{
			&rbacv1.PolicyRule{
				ApiGroups: []string{""},
				Resources: []string{"endpoints", "services", "pods", "namespaces"},
				Verbs:     []string{"list", "watch"},
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, cr); err != nil {
		return maskAny(err)
	}

	// Create system:coredns cluster role binding
	crb := &rbacv1.ClusterRoleBinding{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String("system:coredns"),
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
		},
		RoleRef: &rbacv1.RoleRef{
			ApiGroup: k8s.String("rbac.authorization.k8s.io"),
			Kind:     k8s.String("ClusterRole"),
			Name:     k8s.String("system:coredns"),
		},
		Subjects: []*rbacv1.Subject{
			&rbacv1.Subject{
				Kind:      k8s.String("ServiceAccount"),
				Name:      k8s.String("coredns"),
				Namespace: k8s.String("kube-system"),
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, crb); err != nil {
		return maskAny(err)
	}

	// Render corefile for coredns
	corefileOpts := struct {
		ClusterDomain         string
		ServiceClusterIPRange string
	}{
		ClusterDomain:         flags.Kubernetes.ClusterDomain,
		ServiceClusterIPRange: flags.Kubernetes.ServiceClusterIPRange,
	}
	corefile, err := util.RenderToString(log, corefileTemplate, corefileOpts)
	if err != nil {
		return maskAny(err)
	}

	// Create coredns config map
	cm := &corev1.ConfigMap{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("coredns"),
			Namespace: k8s.String("kube-system"),
		},
		Data: map[string]string{
			"Corefile": corefile,
		},
	}
	if err := util.CreateOrUpdate(ctx, client, cm); err != nil {
		return maskAny(err)
	}

	// Create coredns deployment
	ds := &appsv1.Deployment{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("coredns"),
			Namespace: k8s.String("kube-system"),
			Labels: map[string]string{
				"k8s-app":            "coredns",
				"kubernetes.io/name": "CoreDNS",
			},
		},
		Spec: &appsv1.DeploymentSpec{
			Replicas: k8s.Int32(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "coredns",
				},
			},
			Strategy: &appsv1.DeploymentStrategy{
				Type: k8s.String("RollingUpdate"),
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: util.IntOrStringI(1),
				},
			},
			Template: &corev1.PodTemplateSpec{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "coredns",
					},
				},
				Spec: &corev1.PodSpec{
					Containers: []*corev1.Container{
						&corev1.Container{
							Name:  k8s.String("coredns"),
							Image: k8s.String(flags.Images.CoreDNS),
							Args:  []string{"-conf", "/etc/coredns/Corefile"},
							VolumeMounts: []*corev1.VolumeMount{
								&corev1.VolumeMount{
									MountPath: k8s.String("/etc/coredns"),
									Name:      k8s.String("config-volume"),
								},
							},
							Ports: []*corev1.ContainerPort{
								&corev1.ContainerPort{
									ContainerPort: k8s.Int32(53),
									Name:          k8s.String("dns"),
									Protocol:      k8s.String("UDP"),
								},
								&corev1.ContainerPort{
									ContainerPort: k8s.Int32(53),
									Name:          k8s.String("dns-tcp"),
									Protocol:      k8s.String("TCP"),
								},
								&corev1.ContainerPort{
									ContainerPort: k8s.Int32(9153),
									Name:          k8s.String("metrics"),
									Protocol:      k8s.String("TCP"),
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: &corev1.Handler{
									HttpGet: &corev1.HTTPGetAction{
										Path:   k8s.String("/health"),
										Port:   util.IntOrStringI(8080),
										Scheme: k8s.String("HTTP"),
									},
								},
								InitialDelaySeconds: k8s.Int32(60),
								TimeoutSeconds:      k8s.Int32(5),
								SuccessThreshold:    k8s.Int32(1),
								FailureThreshold:    k8s.Int32(5),
							},
						},
					},
					DnsPolicy:          k8s.String("Default"),
					ServiceAccountName: k8s.String("coredns"),
					Volumes: []*corev1.Volume{
						&corev1.Volume{
							Name: k8s.String("config-volume"),
							VolumeSource: &corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: &corev1.LocalObjectReference{
										Name: k8s.String("coredns"),
									},
									Items: []*corev1.KeyToPath{
										&corev1.KeyToPath{
											Key:  k8s.String("Corefile"),
											Path: k8s.String("Corefile"),
										},
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

	// Create coredns service
	svc := &corev1.Service{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("kube-dns"),
			Namespace: k8s.String("kube-system"),
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
			Labels: map[string]string{
				"k8s-app":                       "coredns",
				"kubernetes.io/cluster-service": "true",
				"kubernetes.io/name":            "CoreDNS",
			},
		},
		Spec: &corev1.ServiceSpec{
			Selector: map[string]string{
				"k8s-app": "coredns",
			},
			ClusterIP: k8s.String(flags.Kubernetes.ClusterDNS),
			Ports: []*corev1.ServicePort{
				&corev1.ServicePort{
					Name:     k8s.String("dns"),
					Port:     k8s.Int32(53),
					Protocol: k8s.String("UDP"),
				},
				&corev1.ServicePort{
					Name:     k8s.String("dns-tcp"),
					Port:     k8s.Int32(53),
					Protocol: k8s.String("TCP"),
				},
			},
		},
	}
	if err := util.CreateOrUpdate(ctx, client, svc); err != nil {
		return maskAny(err)
	}

	return nil
}
