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

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

var _ = Describe("PodDistribution Controller with Distribution feature", func() {
	Context("with .spec.distribution", func() {
		const resourceName = "test-resource"

		ctx := logr.NewContext(context.Background(), testEnvLogger)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		deployment := &appsv1.Deployment{}

		BeforeEach(func() {
			By("creating the target deployment for the PodDistribution")
			err := k8sClient.Get(ctx, typeNamespacedName, deployment)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, targetDeployment(resourceName))).To(Succeed())
			}

			By("creating the custom resource for the Kind PodDistribution")
			resource := &poddistributionv1alpha1.PodDistribution{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil && errors.IsNotFound(err) {
				affinity := &corev1.PodAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								TopologyKey: "topo-key",
							},
						},
					},
				}
				antiAffinity := &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								TopologyKey: "topo-key",
							},
						},
					},
				}

				resource = newTestPodDistribution(
					resourceName,
					WithSelector("Deployment", testLabels()),
					WithPodAffinity(affinity),
					WithPodAntiAffinity(antiAffinity),
					WithNodeName("node-sample"),
					WithNodeSelector(map[string]string{"hostname": "node-sample"}),
				)
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("reconciling the custom resource for the Kind PodDistribution")
			controllerReconciler := &PodDistributionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
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

		Describe(".affinity", func() {

			Describe(".affinity", func() {
				It("check target deployment has pod affinity", func() {
					dep := &appsv1.Deployment{}
					Eventually(func() error {
						return k8sClient.Get(ctx, typeNamespacedName, dep)
					}).WithTimeout(10 * time.Second).Should(BeNil())

					Expect(dep.Spec.Template.Spec.Affinity.PodAffinity).ShouldNot(BeNil())
					Expect(dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution).ShouldNot(BeNil())
					Expect(len(dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).Should(BeIdenticalTo(1))
				})
			})

			Describe(".antiAffinity", func() {
				It("check target deployment has pod anti affinity", func() {
					dep := &appsv1.Deployment{}
					Eventually(func() error {
						return k8sClient.Get(ctx, typeNamespacedName, dep)
					}).WithTimeout(10 * time.Second).Should(BeNil())

					Expect(dep.Spec.Template.Spec.Affinity.PodAntiAffinity).ShouldNot(BeNil())
					Expect(dep.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution).ShouldNot(BeNil())
					Expect(len(dep.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).Should(BeIdenticalTo(1))
				})
			})
		})

		Describe(".node", func() {
			Describe(".name", func() {
				It("check target deployment has nodeName", func() {
					dep := &appsv1.Deployment{}
					Eventually(func() error {
						return k8sClient.Get(ctx, typeNamespacedName, dep)
					}).WithTimeout(10 * time.Second).Should(BeNil())

					Expect(dep.Spec.Template.Spec.NodeName).Should(BeIdenticalTo("node-sample"))
				})

			})

			Describe(".selector", func() {
				It("check target deployment has nodeSelector", func() {
					dep := &appsv1.Deployment{}
					Eventually(func() error {
						return k8sClient.Get(ctx, typeNamespacedName, dep)
					}).WithTimeout(10 * time.Second).Should(BeNil())

					Expect(dep.Spec.Template.Spec.NodeSelector).ShouldNot(BeNil())
					Expect(len(dep.Spec.Template.Spec.NodeSelector)).Should(BeIdenticalTo(1))
					Expect(dep.Spec.Template.Spec.NodeSelector["hostname"]).Should(BeIdenticalTo("node-sample"))
				})

			})
		})
	})
})
