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

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodDistribution Webhook", func() {
	Context("When creating PodDistribution under Validating Webhook", func() {
		Context("With .spec.minAvailable.Policy = 3", func() {
			It("Should deny the fixed value in .spec.minAvailable.Policy", func() {
				pd := &PodDistribution{
					Spec: PodDistributionSpec{
						MinAvailable: &PodDistributionMinAvailableSpec{
							Policy: "3",
						},
					},
				}
				_, err := pd.ValidateCreate()
				Expect(err).Should(HaveOccurred())
			})
		})
		Context("With .spec.minAvailable.Policy = <unknown-format>", func() {
			It("Should deny the fixed value in .spec.minAvailable.Policy", func() {
				pd := &PodDistribution{
					Spec: PodDistributionSpec{
						MinAvailable: &PodDistributionMinAvailableSpec{
							Policy: "unknown-policy",
						},
					},
				}
				_, err := pd.ValidateCreate()
				Expect(err).Should(HaveOccurred())
			})
		})

		It("Should admit if all required fields are provided", func() {
			pd := &PodDistribution{
				Spec: PodDistributionSpec{
					MinAvailable: &PodDistributionMinAvailableSpec{
						Policy: "50%",
					},
				},
			}
			_, err := pd.ValidateCreate()
			Expect(err).Should(BeNil())

		})
	})

	Context("When updating PodDistribution under Validating Webhook", func() {
		Context("With .spec.minAvailable.Policy = 3", func() {
			It("Should deny the fixed value in .spec.minAvailable.Policy", func() {
				pd := &PodDistribution{
					Spec: PodDistributionSpec{
						MinAvailable: &PodDistributionMinAvailableSpec{
							Policy: "3",
						},
					},
				}
				_, err := pd.ValidateUpdate(nil)
				Expect(err).Should(HaveOccurred())
			})
		})
		Context("With .spec.minAvailable.Policy = <unknown-format>", func() {
			It("Should deny the fixed value in .spec.minAvailable.Policy", func() {
				pd := &PodDistribution{
					Spec: PodDistributionSpec{
						MinAvailable: &PodDistributionMinAvailableSpec{
							Policy: "unknown-policy",
						},
					},
				}
				_, err := pd.ValidateUpdate(nil)
				Expect(err).Should(HaveOccurred())
			})
		})

		It("Should admit if all required fields are provided", func() {
			pd := &PodDistribution{
				Spec: PodDistributionSpec{
					MinAvailable: &PodDistributionMinAvailableSpec{
						Policy: "50%",
					},
				},
			}
			_, err := pd.ValidateUpdate(nil)
			Expect(err).Should(BeNil())

		})
	})
})
