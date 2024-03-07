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
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var poddistributionlog = logf.Log.WithName("poddistribution-resource")
var podDistributionWebhookK8sClient client.Client

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PodDistribution) SetupWebhookWithManager(mgr ctrl.Manager) error {
	podDistributionWebhookK8sClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Validator = &PodDistribution{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PodDistribution) ValidateCreate() (admission.Warnings, error) {
	poddistributionlog.Info("validate create", "name", r.Name)

	// TODO: .spec.maxUnavailable
	if r.Spec.MinAvailable == nil {
		return nil, fmt.Errorf(".spec.minAvailable or .spec.maxUnavailable must be specified")
	}

	if r.Spec.MinAvailable != nil {
		if err := validateMinAvailableSpec(r.Spec.MinAvailable); err != nil {
			return nil, err
		}
	}

	poddistributionlog.Info("passed the validation webhook", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PodDistribution) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	poddistributionlog.Info("validate update", "name", r.Name)

	poddistributionlog.Info("passed the validation webhook", "name", r.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PodDistribution) ValidateDelete() (admission.Warnings, error) {
	poddistributionlog.Info("validate delete", "name", r.Name)

	poddistributionlog.Info("passed the validation webhook", "name", r.Name)
	return nil, nil
}

func validateMinAvailableSpec(minAvailable *PodDistributionMinAvailableSpec) error {
	if err := validateMinAvailableSpecPolicy(minAvailable.Policy); err != nil {
		return err
	}
	if n, err := strconv.Atoi(minAvailable.Policy); err == nil {
		return fmt.Errorf(".spec.minAvailable.policy deny the fixed value %d", n)
	}

	if strings.Contains(minAvailable.Policy, "%") {
		if minAvailable.Policy[len(minAvailable.Policy)-1] != '%' {
			return fmt.Errorf(".spec.minAvailable.policy with ratio must be formatted as '.*%%'")
		}

		if _, err := strconv.Atoi(minAvailable.Policy[:len(minAvailable.Policy)-1]); err != nil {
			return fmt.Errorf(".spec.minAvailable.policy with ratio must be formatted as '.*%%'")
		}
	}

	return nil
}

func validateMinAvailableSpecPolicy(policy string) error {
	if n, err := strconv.Atoi(policy); err == nil {
		return fmt.Errorf(".spec.minAvailable.policy deny the fixed value %d", n)
	}

	if strings.Contains(policy, "%") {
		if policy[len(policy)-1] != '%' {
			return fmt.Errorf(".spec.minAvailable.policy with ratio must be formatted as '.*%%'")
		}

		if _, err := strconv.Atoi(policy[:len(policy)-1]); err != nil {
			return fmt.Errorf(".spec.minAvailable.policy with ratio must be formatted as '.*%%'")
		}

		return nil
	}

	// TODO: validate policy name
	return fmt.Errorf(".spec.minAvailable.policy with unknown format '%s'must be denied", policy)
}
