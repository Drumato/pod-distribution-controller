/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

var _ = Describe("PodDistribution Controller with PDB feature", func() {
	Context("with .spec.distribution", func() {
		const resourceName = "test-resource"

		ctx := logr.NewContext(context.Background(), testEnvLogger)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		deployment := &appsv1.Deployment{}
		poddistribution := &poddistributionv1alpha1.PodDistribution{}

		BeforeEach(func() {
			By("creating the target deployment for the PodDistribution")
			err := k8sClient.Get(ctx, typeNamespacedName, deployment)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, targetDeployment(resourceName))).To(Succeed())
			}

			By("creating the custom resource for the Kind PodDistribution")
			err = k8sClient.Get(ctx, typeNamespacedName, poddistribution)
			if err != nil && errors.IsNotFound(err) {
				resource := &poddistributionv1alpha1.PodDistribution{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: poddistributionv1alpha1.PodDistributionSpec{
						Selector: poddistributionv1alpha1.PodDistributionSelector{
							Kind: "Deployment",
							LabelSelector: metav1.LabelSelector{
								MatchLabels: testLabels(),
							},
						},
						Distribution: &poddistributionv1alpha1.DistributionSpec{
							Pod: poddistributionv1alpha1.DistributionPodSpec{
								Affinity: &poddistributionv1alpha1.DistributionPodAffinitySpec{
									Manual: &corev1.PodAffinity{
										PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
											{
												Weight: 100,
												PodAffinityTerm: corev1.PodAffinityTerm{
													TopologyKey: "topo-key",
												},
											},
										},
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance PodDistribution")
			resource := &poddistributionv1alpha1.PodDistribution{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the target deployment")
			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, dep)
			}).WithTimeout(10 * time.Second).Should(BeNil())
			Expect(k8sClient.Delete(ctx, dep)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PodDistributionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("try to get the reconcile target")
			resource := &poddistributionv1alpha1.PodDistribution{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, resource)
			}).WithTimeout(10 * time.Second).Should(BeNil())
			Expect(len(resource.Status.TargetPodCollections)).Should(BeIdenticalTo(1))

			By("try to get the target deployment resource")
			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, dep)
			}).WithTimeout(10 * time.Second).Should(BeNil())

			By("check the affinity field")
			Expect(dep.Spec.Template.Spec.Affinity.PodAffinity).ShouldNot(BeNil())
			Expect(dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution).ShouldNot(BeNil())
			Expect(len(dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).Should(BeIdenticalTo(1))
		})
	})
})
