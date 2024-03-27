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

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

var _ = Describe("PodDistribution Controller with Labeler feature", func() {
	Context("with .spec.labeler", func() {
		const resourceName = "test-resource"

		ctx := logr.NewContext(context.Background(), testEnvLogger)
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			By("creating the target deployment for the PodDistribution")
			deployment := targetDeployment(resourceName)
			err := k8sClient.Get(ctx, typeNamespacedName, deployment)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, targetDeployment(resourceName))).To(Succeed())
			}

			By("creating the custom resource for the Kind PodDistribution")
			var resource *poddistributionv1alpha1.PodDistribution
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil && errors.IsNotFound(err) {
				ls := &poddistributionv1alpha1.LabelerSpec{
					LabelRules: []poddistributionv1alpha1.LabelRule{
						{
							Kind:          "Deployment",
							CELExpression: "true",
						},
					},
				}
				resource = newTestPodDistribution(
					resourceName,
					WithSelector("Deployment", testLabels()),
					WithLabeler(ls),
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

			AfterEach(func() {
				resource := &poddistributionv1alpha1.PodDistribution{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())

				By("Cleanup the specific resource instance PodDistribution")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

				By("Cleanup the target deployment for the PodDistribution")
				Eventually(func() error {
					deployment := &appsv1.Deployment{}
					err := k8sClient.Get(ctx, typeNamespacedName, deployment)
					if err != nil {
						return err
					}
					return k8sClient.Delete(ctx, deployment)
				}).Should(Succeed())
			})

			It("should create the Labeler", func() {
				labeler := &poddistributionv1alpha1.Labeler{}
				Eventually(func() error {
					return k8sClient.Get(ctx, typeNamespacedName, labeler)
				}).WithTimeout(10 * time.Second).Should(BeNil())
				Expect(labeler.Spec.LabelRules).ShouldNot(BeNil())
				Expect(len(labeler.Spec.LabelRules)).Should(BeIdenticalTo(1))
				Expect(labeler.Spec.LabelRules[0].CELExpression).Should(BeIdenticalTo("true"))
			})
		})

	})
})
