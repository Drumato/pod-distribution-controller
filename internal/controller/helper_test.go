package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

type TestPodDistributionOption func(pd *poddistributionv1alpha1.PodDistribution)

func newTestPodDistribution(name string, opts ...TestPodDistributionOption) *poddistributionv1alpha1.PodDistribution {
	pd := &poddistributionv1alpha1.PodDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: poddistributionv1alpha1.PodDistributionSpec{},
	}

	for _, opt := range opts {
		opt(pd)
	}

	return pd
}

func WithSelector(kind string, matchLabels map[string]string) TestPodDistributionOption {
	return func(pd *poddistributionv1alpha1.PodDistribution) {
		pd.Spec.Selector = poddistributionv1alpha1.PodDistributionSelector{
			Kind: kind,
			LabelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		}
	}
}

func WithPDBMinAvailable(minAvailable *poddistributionv1alpha1.PodDistributionMinAvailableSpec) TestPodDistributionOption {
	return func(pd *poddistributionv1alpha1.PodDistribution) {
		if pd.Spec.PDB == nil {
			pd.Spec.PDB = &poddistributionv1alpha1.PodDistributionPDBSpec{}
		}

		pd.Spec.PDB.MinAvailable = minAvailable
	}
}

func WithPodAffinity(affinity *corev1.PodAffinity) TestPodDistributionOption {
	return func(pd *poddistributionv1alpha1.PodDistribution) {
		if pd.Spec.Distribution == nil {
			pd.Spec.Distribution = &poddistributionv1alpha1.DistributionSpec{}
		}

		if pd.Spec.Distribution.Pod.Affinity == nil {
			pd.Spec.Distribution.Pod.Affinity = &poddistributionv1alpha1.DistributionPodAffinitySpec{}
		}
		pd.Spec.Distribution.Pod.Affinity.Manual = affinity
	}
}

func WithPodAntiAffinity(antiAffinity *corev1.PodAntiAffinity) TestPodDistributionOption {
	return func(pd *poddistributionv1alpha1.PodDistribution) {
		if pd.Spec.Distribution == nil {
			pd.Spec.Distribution = &poddistributionv1alpha1.DistributionSpec{}
		}

		if pd.Spec.Distribution.Pod.AntiAffinity == nil {
			pd.Spec.Distribution.Pod.AntiAffinity = &poddistributionv1alpha1.DistributionPodAntiAffinitySpec{}
		}
		pd.Spec.Distribution.Pod.AntiAffinity.Manual = antiAffinity
	}
}

func targetDeployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    testLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(5)),
			Selector: &metav1.LabelSelector{
				MatchLabels: testLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: testLabels(),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}
}

func testLabels() map[string]string {
	return map[string]string{
		"app": "test",
	}
}
