package operator

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubesphere.io/fluentbit-operator/api/v1alpha2"
)

func MakeRBACObjects(fbName, fbNamespace string) (rbacv1.ClusterRole, corev1.ServiceAccount, rbacv1.ClusterRoleBinding) {
	cr := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
		},
	}

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fbName,
			Namespace: fbNamespace,
		},
	}

	crb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fbName,
				Namespace: fbNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "fluent-bit",
		},
	}

	return cr, sa, crb
}

func MakeDaemonSet(fb v1alpha2.FluentBit, logPath string) appsv1.DaemonSet {
	ds := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fb.Name,
			Namespace: fb.Namespace,
			Labels:    fb.Labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: fb.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fb.Name,
					Namespace: fb.Namespace,
					Labels:    fb.Labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fb.Name,
					Volumes: []corev1.Volume{
						{
							Name: "varlibcontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: logPath,
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fb.Spec.FluentBitConfigName,
								},
							},
						},
						{
							Name: "varlogs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name:         "positions",
							VolumeSource: fb.Spec.PositionDB,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "fluent-bit",
							Image:           fb.Spec.Image,
							ImagePullPolicy: fb.Spec.ImagePullPolicy,
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 2020,
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "varlibcontainers",
									ReadOnly:  true,
									MountPath: logPath,
								},
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/fluent-bit/config",
								},
								{
									Name:      "varlogs",
									ReadOnly:  true,
									MountPath: "/var/log/",
								},
								{
									Name:      "positions",
									MountPath: "/fluent-bit/tail",
								},
							},
						},
					},
					Tolerations: fb.Spec.Tolerations,
				},
			},
		},
	}

	// Mount Secrets
	for _, secret := range fb.Spec.Secrets {
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: secret,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret,
				},
			},
		})
		ds.Spec.Template.Spec.Containers[0].VolumeMounts = append(ds.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      secret,
			ReadOnly:  true,
			MountPath: fmt.Sprintf("/fluent-bit/secrets/%s", secret),
		})
	}

	return ds
}
