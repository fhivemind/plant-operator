/*
Copyright 2023.

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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv1 "github.com/fhivemind/plant-operator/api/v1"
)

//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=get;list;watch
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete

// PlantReconciler reconciles a Plant object
type PlantReconciler struct {
	Client client.Client // differentiate Client and PlantReconciler calls
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Plant{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(logPredicate())).
		Owns(&corev1.Service{}, builder.WithPredicates(logPredicate())).
		Owns(&networkingv1.Ingress{}, builder.WithPredicates(logPredicate())).
		Complete(r)
}

// Reconcile ensures that Plant and its owned resources match the required states
// from the cluster. This method will fetch the controlled Plant object, ensure
// that Plant has Finalizers and required states to safely address deletion, and
// finally, invoke StateHandle to perform control tasks based on the requirements.
func (r *PlantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling")

	// Fetch plant resource if it exists
	plant := &apiv1.Plant{}
	if err := r.Client.Get(ctx, req.NamespacedName, plant); err != nil {
		logger.Info("Deleted successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizers to plant if missing
	if controllerutil.AddFinalizer(plant, apiv1.Finalizer) {
		// Since Update will trigger reconcile, there is no need to finish this request.
		// If it fails, error handler will reschedule it anyway
		if err := r.Client.Update(ctx, plant); err != nil {
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant after adding finalizers: %w", err))
		}
		logger.Info("Finalizer added, rescheduling")
		return ctrl.Result{}, nil
	}

	// Check if plant scheduled but not configured for deletion
	if !plant.DeletionTimestamp.IsZero() && plant.Status.State != apiv1.StateDeleting {
		if err := r.UpdateState(ctx, plant, apiv1.StateDeleting); err != nil {
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant status after triggering deletion: %w", err))
		}
		logger.Info("Marked Plant for deletion")
	}

	// Run main control loop
	requeue, err := r.StateHandle(ctx, plant)
	if err != nil {
		return r.ErrorHandle(ctx, plant, fmt.Errorf("could not handle Plant control loop: %w", err))
	}
	return ctrl.Result{Requeue: requeue}, nil
}

// ErrorHandle logs the error, puts Plant into apiv1.StateError state, and reschedules the request.
func (r *PlantReconciler) ErrorHandle(ctx context.Context, plant *apiv1.Plant, err error) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, "Error occurred, rescheduling")
	return ctrl.Result{Requeue: true}, r.UpdateState(ctx, plant, apiv1.StateError)
}

// StateHandle runs the main control loop based on the configured Plant state.
// Returns true if reconcile should be triggered.
func (r *PlantReconciler) StateHandle(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	logger := log.FromContext(ctx)

	switch state := plant.Status.State; state {
	case apiv1.StateProcessing, apiv1.StateError: // Keep processing Plant until "Ready"
		logger.Info(fmt.Sprintf("Handling %s state", state))
		return r.HandleProcessingState(ctx, plant)

	case apiv1.StateDeleting: // Keep deleting until the Plant is gone
		logger.Info(fmt.Sprintf("Handling %s state", state))
		return r.HandleDeletingState(ctx, plant)

	default: // Reprocess since Plant is in "Ready" or unknown state
		logger.Info("Marked for processing, rescheduling")
		return true, r.UpdateState(ctx, plant, apiv1.StateProcessing)
	}
}

// HandleProcessingState processes all child resources by ensuring that they are in proper states.
// Check doHandleWith to get more details on how child resource execution is handled.
// Returns true if reconcile should be triggered. Updates Status with observed results.
func (r *PlantReconciler) HandleProcessingState(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	logger := log.FromContext(ctx)

	// do processing
	deployment := &appsv1.Deployment{}
	service := &corev1.Service{}
	ingress := &networkingv1.Ingress{}

	var errGroup errgroup.Group
	errGroup.Go(func() error { return doHandleWith(ctx, plant, deployment, r.deploymentHandler(ctx, plant)) })
	errGroup.Go(func() error { return doHandleWith(ctx, plant, service, r.serviceManager(ctx, plant)) })
	errGroup.Go(func() error { return doHandleWith(ctx, plant, ingress, r.ingressManager(ctx, plant)) })

	processingErr := errGroup.Wait()

	// update states
	newState := plant.DetermineState()
	if newState == apiv1.StateReady && plant.Status.State != apiv1.StateReady {
		logger.Info("All tasks done, setting Ready state")
	} else {
		logger.Info("Tasks are not yet in Ready state, rescheduling")
	}
	stateErr := r.UpdateState(ctx, plant, newState)
	if stateErr != nil {
		stateErr = fmt.Errorf("error while updating Plant status during processing: %w", stateErr)
	}
	return newState != apiv1.StateReady, errors.Join(processingErr, stateErr)
}

// HandleDeletingState remove all hanging resources. The garbage collector will
// handle internal resource deletion, while we handle the external ones here.
func (r *PlantReconciler) HandleDeletingState(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	// Nothing to do here for now, continue

	// Remove finalizers to notify that deletion is completed
	if controllerutil.RemoveFinalizer(plant, apiv1.Finalizer) {
		if err := r.Client.Update(ctx, plant); err != nil {
			return false, fmt.Errorf("could not update Plant after removing finalizers: %w", err)
		}
	}
	return false, nil
}

func (r *PlantReconciler) UpdateState(ctx context.Context, plant *apiv1.Plant, newState apiv1.State) error {
	plant.Status.State = newState
	return r.UpdateStatus(ctx, plant)
}

func (r *PlantReconciler) UpdateStatus(ctx context.Context, plant *apiv1.Plant) error {
	if plant.Status.LastUpdateTime == nil {
		plant.Status.LastUpdateTime = new(metav1.Time)
	}
	*plant.Status.LastUpdateTime = metav1.Now()
	if err := r.Client.Status().Update(ctx, plant); err != nil {
		return fmt.Errorf("could not update Plant status: %w", err)
	}
	return nil
}

// doHandleWith handles subresouce configuration and updates plant with the execution results.
// It relies on dynamic resource.Handler to perform operations.
func doHandleWith[T client.Object](ctx context.Context, plant *apiv1.Plant, obj T, handler resource.Handler[T]) error {
	// Define initial states
	resourceState := apiv1.StateProcessing
	conditionState := metav1.ConditionFalse

	// Run handler and extract results
	flow, err := handler.Handle(ctx, obj)
	if err != nil { // generic fail
		resourceState = apiv1.StateError
	} else if flow.Done() { // finished with success
		resourceState = apiv1.StateReady
		conditionState = metav1.ConditionTrue
	}

	// Update plant resource conditions
	plant.UpdateCondition(apiv1.ConditionType(handler.Name), conditionState, flow.OperationName(),
		fmt.Sprintf("%s operation is in %s state", flow.OperationName(), resourceState))

	// Update plant resource status
	result := apiv1.ResourceStatus{
		Name:  handler.Name,
		GVK:   obj.GetObjectKind().GroupVersionKind().String(),
		State: resourceState,
	}

	found := false
	for id, item := range plant.Status.Resources {
		if item.Name == result.Name {
			found = true
			plant.Status.Resources[id] = result
			break
		}
	}
	if !found {
		plant.Status.Resources = append(plant.Status.Resources, result)
	}

	// Pass result once everything has been handled
	return err
}
