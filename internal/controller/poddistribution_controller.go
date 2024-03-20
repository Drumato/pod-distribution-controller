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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	policyv1 "k8s.io/api/policy/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	applypolicyv1 "k8s.io/client-go/applyconfigurations/policy/v1"
)

const (
	defaultFieldManager = "pod-distribution-controller"
)

// PodDistributionReconciler reconciles a PodDistribution object
type PodDistributionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=poddistributions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=poddistributions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=poddistributions/finalizers,verbs=update
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodDistributionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	logger.V(6).Info("start reconcile")

	pd := &poddistributionv1alpha1.PodDistribution{}
	if err := r.Get(ctx, req.NamespacedName, pd); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.V(0).Error(err, "failed to get poddistribution")
		return ctrl.Result{}, err
	}

	if pd.DeletionTimestamp != nil {
		logger.V(6).Info("the resource is being deleted. reconcile will be canceled")
		return ctrl.Result{}, nil
	}

	podCollections, err := r.listTargetPodCollections(ctx, pd)
	if err != nil {
		logger.V(0).Error(err, "error in listTargetPodCollections()")
		return ctrl.Result{}, err
	}

	pd.Status.TargetPodCollections = podCollections

	if err := r.reconcilePodDisruptionBudget(ctx, logger, pd); err != nil {
		logger.V(0).Error(err, "error in reconcilePodDisruptionBudget()")
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, pd); err != nil {
		logger.V(0).Error(err, "error in r.Status().Update()")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PodDistributionReconciler) reconcilePodDisruptionBudget(
	ctx context.Context,
	logger logr.Logger,
	pd *poddistributionv1alpha1.PodDistribution,
) error {
	if pd.Spec.PDB.MinAvailable == nil {
		// TODO: maxUnavailable not implemented
		return nil
	}

	// check whether a PodDistribution denies the request that may violate undrainable policy.
	violateUndrainPolicyError := r.detectMinAvailableUndrainablePolicy(logger, pd)
	if !pd.Spec.AllowAugmentDeploymentReplicas && !pd.Spec.PDB.MinAvailable.AllowUndrainable {
		if violateUndrainPolicyError != nil {
			return violateUndrainPolicyError
		}
	}

	if pd.Spec.AllowAugmentDeploymentReplicas {
		// TODO: augment the replicas field of the target deployment and update it.
		/*
			if err := r.augmentTargetDeploymentReplica(); err != nil {
				return err
			}
		*/
	}

	labelSelectorRequirements := make([]*applymetav1.LabelSelectorRequirementApplyConfiguration, len(pd.Spec.Selector.LabelSelector.MatchExpressions))
	for i := range pd.Spec.Selector.LabelSelector.MatchExpressions {
		expr := pd.Spec.Selector.LabelSelector.MatchExpressions[i]
		labelSelectorRequirements[i] = applymetav1.LabelSelectorRequirement().WithKey(expr.Key).WithValues(expr.Values...).WithOperator(expr.Operator)
	}
	pdb := applypolicyv1.PodDisruptionBudget(pd.Name, pd.Namespace).
		WithSpec(
			applypolicyv1.PodDisruptionBudgetSpec().WithSelector(
				applymetav1.LabelSelector().WithMatchLabels(pd.Spec.Selector.LabelSelector.MatchLabels).WithMatchExpressions(labelSelectorRequirements...)).WithMinAvailable(intstr.FromString(pd.Spec.PDB.MinAvailable.Policy)),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pdb)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current policyv1.PodDisruptionBudget
	err = r.Get(ctx, client.ObjectKey{Namespace: pd.Namespace, Name: pd.Name}, &current)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := applypolicyv1.ExtractPodDisruptionBudget(&current, defaultFieldManager)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(pdb, currApplyConfig) {
		return nil
	}

	if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: defaultFieldManager,
		Force:        ptr.To(true),
	}); err != nil {
		return err
	}

	return nil
}

func (r *PodDistributionReconciler) listTargetPodCollections(
	ctx context.Context,
	pd *poddistributionv1alpha1.PodDistribution,
) ([]poddistributionv1alpha1.TargetPodCollection, error) {
	switch pd.Spec.Selector.Kind {
	case poddistributionv1alpha1.PodDistributionSelectorKindDeployment:
		deployments, err := r.listTargetDeployments(ctx, pd)
		if err != nil {
			return nil, err
		}

		collections := make([]poddistributionv1alpha1.TargetPodCollection, len(deployments.Items))
		for i := range deployments.Items {
			collections[i] = poddistributionv1alpha1.TargetPodCollection{
				Kind:      pd.Spec.Selector.Kind,
				Name:      deployments.Items[i].Name,
				Namespace: deployments.Items[i].Namespace,
				Replicas:  *deployments.Items[i].Spec.Replicas,
			}
		}
		return collections, nil
	default:
		return nil, fmt.Errorf("%s not unsupported", pd.Spec.Selector.Kind)
	}
}

func (r *PodDistributionReconciler) listTargetDeployments(
	ctx context.Context,
	pd *poddistributionv1alpha1.PodDistribution,
) (*appsv1.DeploymentList, error) {
	deployments := &appsv1.DeploymentList{}

	if err := r.List(ctx, deployments, &client.ListOptions{Namespace: pd.Namespace}); err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&pd.Spec.Selector.LabelSelector)
	if err != nil {
		return nil, err
	}

	filtered := &appsv1.DeploymentList{}
	for _, d := range deployments.Items {
		if !selector.Matches(labels.Set(d.ObjectMeta.Labels)) {
			continue
		}
		filtered.Items = append(filtered.Items, d)
	}

	return filtered, nil
}

func (r *PodDistributionReconciler) detectMinAvailableUndrainablePolicy(
	logger logr.Logger,
	pd *poddistributionv1alpha1.PodDistribution,
) error {
	for _, collection := range pd.Status.TargetPodCollections {
		pv := intstr.FromString(pd.Spec.PDB.MinAvailable.Policy)
		expectedReplicas, err := intstr.GetScaledValueFromIntOrPercent(&pv, int(collection.Replicas), true)
		if err != nil {
			return err
		}

		logger.V(0).Info(".spec.pdb.minAvailable.Policy scaled up", "value", expectedReplicas)

		if int(collection.Replicas) <= expectedReplicas {
			return fmt.Errorf("undrainable policy detected: %s replicas must be bigger than .spec.pdb.minAvailable actual value %d", collection.Name, expectedReplicas)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodDistributionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&poddistributionv1alpha1.PodDistribution{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Complete(r)
}
