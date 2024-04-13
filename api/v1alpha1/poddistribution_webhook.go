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
	// "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var poddistributionlog = logf.Log.WithName("poddistribution-resource")

//var podDistributionWebhookK8sClient client.Client

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PodDistribution) SetupWebhookWithManager(mgr ctrl.Manager) error {
	//podDistributionWebhookK8sClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Validator = &PodDistribution{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PodDistribution) ValidateCreate() (admission.Warnings, error) {
	poddistributionlog.Info("validate create", "name", r.Name)

	if err := r.validatePDBSpec(); err != nil {
		return nil, err
	}

	poddistributionlog.Info("passed the validation webhook", "name", r.Name)
	return nil, nil
}

func (r *PodDistribution) validatePDBSpec() error {
	// TODO: .spec.maxUnavailable
	if r.Spec.PDB.MinAvailable == nil {
		return fmt.Errorf(".spec.minAvailable or .spec.maxUnavailable must be specified")
	}

	if r.Spec.PDB.MinAvailable != nil {
		if err := validateMinAvailableSpec(r.Spec.PDB.MinAvailable); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PodDistribution) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	poddistributionlog.Info("validate update", "name", r.Name)

	if err := r.validatePDBSpec(); err != nil {
		return nil, err
	}

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
	if err := validatePDBPolicy(".spec.minAvailable.policy" /* field name */, minAvailable.Policy); err != nil {
		return err
	}

	return nil
}

func validatePDBPolicy(fieldName string, policy string) error {
	if n, err := strconv.Atoi(policy); err == nil {
		return fmt.Errorf("%s deny the fixed value %d", fieldName, n)
	}

	if strings.Contains(policy, "%") {
		if policy[len(policy)-1] != '%' {
			return fmt.Errorf("%s with ratio must be formatted as '.*%%'", fieldName)
		}

		if _, err := strconv.Atoi(policy[:len(policy)-1]); err != nil {
			return fmt.Errorf("%s with ratio must be formatted as '.*%%'", fieldName)
		}

		return nil
	}

	return fmt.Errorf("%s with unknown format '%s'", fieldName, policy)
}
