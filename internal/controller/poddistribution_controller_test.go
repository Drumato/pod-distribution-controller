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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
)

var _ = Describe("PodDistribution Controller", func() {
	Context("When reconciling a resource in valid situations", func() {
		const resourceName = "test-resource"

		ctx := logr.NewContext(context.Background(), testEnvLogger)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		Context("with .spec.MinAvailable", func() {
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
							MinAvailable: &poddistributionv1alpha1.PodDistributionMinAvailableSpec{
								Policy: "50%",
							},
						},
					}
					Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				}
			})

			AfterEach(func() {
				resource := &poddistributionv1alpha1.PodDistribution{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())

				By("Cleanup the specific resource instance PodDistribution")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
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
				Expect(len(resource.Status.TargetDeployments)).Should(BeIdenticalTo(1))

				By("try to get the corresponding PodDisruptionBudget resource")
				pdb := &policyv1.PodDisruptionBudget{}
				Eventually(func() error {
					return k8sClient.Get(ctx, typeNamespacedName, pdb)
				}).WithTimeout(10 * time.Second).Should(BeNil())
				Expect(pdb.Spec.MinAvailable.StrVal).Should(BeIdenticalTo("50%"))
			})
		})
	})
})

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
