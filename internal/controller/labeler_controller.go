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
	"encoding/json"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	poddistributionv1alpha1 "github.com/Drumato/pod-distribution-controller/api/v1alpha1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker"
	"github.com/google/cel-go/ext"
	"github.com/google/cel-go/interpreter"

	"google.golang.org/protobuf/types/known/structpb"
)

// LabelerReconciler reconciles a Labeler object
type LabelerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type CELMatchedObject struct {
	Kind          string
	Name          string
	CELExpression string
	JSON          map[string]any
}

//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=labelers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=labelers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=poddistribution.drumato.com,resources=labelers/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LabelerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	logger.V(6).Info("start reconcile")

	l := &poddistributionv1alpha1.Labeler{}
	if err := r.Get(ctx, req.NamespacedName, l); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.V(0).Error(err, "failed to get poddistribution")
		return ctrl.Result{}, err
	}

	if l.DeletionTimestamp != nil {
		logger.V(6).Info("the resource is being deleted. reconcile will be canceled")
		return ctrl.Result{}, nil
	}

	targets, err := r.collectTarget(ctx, l)
	if err != nil {
		logger.V(0).Error(err, "error in collectTarget()")
		return ctrl.Result{}, err
	}

	filtered, err := r.filterWithCELExpression(ctx, targets)
	if err != nil {
		logger.V(0).Error(err, "error in filterWithCELExpression()")
		return ctrl.Result{}, err
	}

	if err := r.updateTargetLabel(ctx, l, filtered); err != nil {
		logger.V(0).Error(err, "error in addLabelToTargets()")
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, l); err != nil {
		logger.V(0).Error(err, "error in r.Status().Update()")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *LabelerReconciler) updateTargetLabel(
	ctx context.Context,
	l *poddistributionv1alpha1.Labeler,
	targets []CELMatchedObject,
) error {
	if l.Status.TargetPodCollections == nil {
		l.Status.TargetPodCollections = []poddistributionv1alpha1.TargetPodCollection{}
	}

	for _, target := range targets {
		switch target.Kind {
		case poddistributionv1alpha1.PodDistributionSelectorKindDeployment:
			// to avoid updating resource that has old status, re-get
			dep := &appsv1.Deployment{}
			if err := r.Client.Get(ctx, types.NamespacedName{Namespace: l.Namespace, Name: target.Name}, dep); err != nil {
				return err
			}

			if dep.Labels == nil {
				dep.Labels = make(map[string]string)
			}

			dep.Labels[poddistributionv1alpha1.PodDistributionManagedByLabel] = l.Name
			if err := r.Client.Update(ctx, dep); err != nil {
				return err
			}

			l.Status.TargetPodCollections = append(l.Status.TargetPodCollections, poddistributionv1alpha1.TargetPodCollection{
				Kind:      poddistributionv1alpha1.PodDistributionSelectorKindDeployment,
				Namespace: l.Namespace,
				Name:      dep.Name,
				Replicas:  *dep.Spec.Replicas,
			})
		default:
			panic("unreachable")
		}
	}

	return nil
}

func celBaseEnvironmentOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		cel.DefaultUTCTimeZone(true),

		cel.CrossTypeNumericComparisons(true),
		cel.OptionalTypes(),

		cel.ASTValidators(
			cel.ValidateDurationLiterals(),
			cel.ValidateTimestampLiterals(),
			cel.ValidateRegexLiterals(),
			cel.ValidateHomogeneousAggregateLiterals(),
		),

		ext.Strings(ext.StringsVersion(2)),
		ext.Sets(),

		cel.CostEstimatorOptions(checker.PresenceTestHasCost(false)),
	}
}

func (r *LabelerReconciler) collectTarget(
	ctx context.Context,
	l *poddistributionv1alpha1.Labeler,
) ([]CELMatchedObject, error) {
	targets := []CELMatchedObject{}

	for _, rule := range l.Spec.LabelRules {
		switch rule.Kind {
		case poddistributionv1alpha1.PodDistributionSelectorKindDeployment:
			list := &appsv1.DeploymentList{}
			if err := r.Client.List(ctx, list); err != nil {
				return nil, err
			}
			for _, dep := range list.Items {
				// to handle the object with cel-go, we marshal/unmarshal the resource
				out, err := json.Marshal(&dep)
				if err != nil {
					return nil, err
				}

				outJSON := map[string]any{}
				if err := json.Unmarshal(out, &outJSON); err != nil {
					return nil, err
				}

				targets = append(targets, CELMatchedObject{
					Kind:          rule.Kind,
					Name:          dep.Name,
					CELExpression: rule.CELExpression,
					JSON:          outJSON,
				})
			}
		default:
			return nil, fmt.Errorf("unknown kind %s", rule.Kind)
		}
	}

	return targets, nil
}

func (r *LabelerReconciler) filterWithCELExpression(
	ctx context.Context,
	targets []CELMatchedObject,
) ([]CELMatchedObject, error) {
	filtered := []CELMatchedObject{}
	celProgramOptions := celBaseProgramOptions()

	for _, target := range targets {
		celEnvOptions := celBaseEnvironmentOptions()
		for k := range target.JSON {
			celEnvOptions = append(celEnvOptions, cel.Variable(k, cel.DynType))
		}
		env, err := cel.NewEnv(celEnvOptions...)
		if err != nil {
			return nil, err
		}
		ast, issues := env.Compile(target.CELExpression)
		if issues != nil {
			return nil, err
		}

		prog, err := env.Program(ast, celProgramOptions...)
		if err != nil {
			return nil, err
		}

		val, _, err := prog.ContextEval(ctx, target.JSON)
		if err != nil {
			return nil, err
		}

		value, err := val.ConvertToNative(reflect.TypeOf(&structpb.Value{}))
		if err != nil {
			return nil, err
		}

		boolV, ok := value.(*structpb.Value)
		if !ok {
			return nil, fmt.Errorf("failed to convert value to bool")
		}

		if !boolV.GetBoolValue() {
			continue
		}

		filtered = append(filtered, target)
	}

	return filtered, nil
}

func celBaseProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{
		cel.EvalOptions(cel.OptOptimize, cel.OptTrackCost),
		cel.CostTrackerOptions(interpreter.PresenceTestHasCost(false)),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LabelerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&poddistributionv1alpha1.Labeler{}).
		Complete(r)
}
