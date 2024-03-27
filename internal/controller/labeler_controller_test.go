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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

var _ = Describe("Labeler Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		labeler := &poddistributionv1alpha1.Labeler{}

		Context("with deployments", func() {
			deploymentNames := []string{
				"dep-1",
				"dep-2",
				"dep-3",
			}

			BeforeEach(func() {
				By("creating deployments")
				for _, name := range deploymentNames {
					if k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: name}, nil) == nil {
						// already created
						continue
					}

					d := &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      name,
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
					err := k8sClient.Create(ctx, d)
					Expect(err).Should(BeNil())
				}

				By("creating the custom resource for the Kind Labeler")
				err := k8sClient.Get(ctx, typeNamespacedName, labeler)
				if err != nil && errors.IsNotFound(err) {
					resource := &poddistributionv1alpha1.Labeler{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: "default",
						},
						Spec: poddistributionv1alpha1.LabelerSpec{
							LabelRules: []poddistributionv1alpha1.LabelRule{
								{
									Kind:          poddistributionv1alpha1.PodDistributionSelectorKindDeployment,
									CELExpression: "true",
								},
							},
						},
					}
					Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				}
			})

			AfterEach(func() {
				resource := &poddistributionv1alpha1.Labeler{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())

				By("Cleanup the specific resource instance Labeler")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

				for _, depName := range deploymentNames {
					dep := &appsv1.Deployment{}
					Eventually(func() error {
						return k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: depName}, dep)
					}).WithTimeout(10 * time.Second).Should(BeNil())
					Expect(k8sClient.Delete(ctx, dep)).To(Succeed())
				}
			})

			It("should successfully reconcile the resource", func() {
				By("Reconciling the created resource")
				controllerReconciler := &LabelerReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("checking labeler target collections")
				Eventually(func() int {
					labeler := &poddistributionv1alpha1.Labeler{}
					err := k8sClient.Get(ctx, typeNamespacedName, labeler)
					if err != nil {
						return 0
					}
					return len(labeler.Status.TargetPodCollections)
				}).WithTimeout(10 * time.Second).Should(BeIdenticalTo(3))
			})

		})
	})
})
