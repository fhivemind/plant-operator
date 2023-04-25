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
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/controllers/workflow"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates/status,verbs=get

// PlantReconciler reconciles a Plant object
type PlantReconciler struct {
	Client   client.Client // differentiate Client and PlantReconciler calls
	Scheme   *runtime.Scheme
	Workflow workflow.Manager
	Recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	bldr := ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Plant{}, builder.WithPredicates(
			notifyWrapper(r.Recorder, predicate.GenerationChangedPredicate{})),
		)

	// add sub-resource trackers
	for _, managedResource := range r.Workflow.Managed() {
		bldr = bldr.Owns(managedResource, builder.WithPredicates(predicate.GenerationChangedPredicate{}))
	}

	return bldr.Complete(r)
}

// Reconcile ensures that Plant and its owned resources match the required states
// from the cluster. This method will fetch the controlled Plant object, ensure
// that Plant has Finalizers and required states to safely address deletion, and
// finally, invoke StateHandle to perform control tasks based on the requirements.
// Reconcile relies on StateHandle to handle the request, and ErrorHandle to
// handle the errors. Request will be rescheduled if an error occurs.
func (r *PlantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling")

	// Fetch plant resource if it exists
	plant := &apiv1.Plant{}
	if err := r.Client.Get(ctx, req.NamespacedName, plant); err != nil {
		logger.Info("Deleted successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is in correct state
	if plant.Status.State != apiv1.StateDeleting {
		// Add finalizers to plant if missing
		if controllerutil.AddFinalizer(plant, apiv1.Finalizer) {
			if err := r.Client.Update(ctx, plant); err != nil {
				return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant after adding finalizers: %w", err))
			}
			logger.Info("Finalizer added to Plant")
		}

		// Set correct state if requested for deletion
		if !plant.DeletionTimestamp.IsZero() {
			if err := r.UpdateStatus(ctx, plant, withState(apiv1.StateDeleting)); err != nil {
				return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant status after triggering deletion: %w", err))
			}
			r.Recorder.Eventf(plant, v1.EventTypeWarning, "Delete", "Marked Plant and its resources for deletion")
		}
	}

	// Execute main control loop
	requeue, err := r.StateHandle(ctx, plant)
	if err != nil {
		return r.ErrorHandle(ctx, plant, fmt.Errorf("could not handle Plant control loop: %w", err))
	}
	return ctrl.Result{Requeue: requeue}, nil
}

// ErrorHandle logs the error, puts Plant into apiv1.StateError state, and returns rescheduled result.
func (r *PlantReconciler) ErrorHandle(ctx context.Context, plant *apiv1.Plant, err error) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, "Error occurred")
	_ = r.UpdateStatus(ctx, plant, withState(apiv1.StateError)) // we ignore this to avoid wrapping the same error
	r.Recorder.Eventf(plant, v1.EventTypeWarning, "SyncWithError", "Reprocessing due to Error: %s", err.Error())
	return ctrl.Result{Requeue: true}, err
}

// StateHandle runs the main control loop based on the configured Plant state.
// Returns true if reconcile should be triggered.
func (r *PlantReconciler) StateHandle(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	logger := log.FromContext(ctx)

	switch state := plant.Status.State; state {
	case apiv1.StateProcessing, apiv1.StateError: // Keep processing Plant until "Ready"
		logger.Info(fmt.Sprintf("Handling Plant %s state", state))
		return r.HandleProcessingState(ctx, plant)

	case apiv1.StateDeleting: // Keep deleting until the Plant is gone
		logger.Info(fmt.Sprintf("Handling Plant %s state", state))
		return r.HandleDeletingState(ctx, plant)

	case apiv1.StateReady: // Reprocess since reconcile was received
		logger.Info("Handling Plant Refresh state")
		return r.HandleProcessingState(ctx, plant)

	default: // Reprocess since Plant is an unknown state
		logger.Info("Marked Plant for Processing, rescheduling")
		return true, r.UpdateStatus(ctx, plant, withState(apiv1.StateProcessing))
	}
}

// HandleProcessingState processes all child resources by ensuring that they are in proper states.
// Check runHandler to get more details on how child resource execution is handled.
// Returns true if reconcile should be triggered. Updates Status with observed results.
func (r *PlantReconciler) HandleProcessingState(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	// Handle workflow
	execResults, execErr := r.Workflow.WithClient(r.Client).Execute(ctx, plant)

	// Update status (with state) since processing updated it
	// We ignore the error as it will be self corrected by the requeue
	uerr := r.UpdateResults(ctx, plant, execResults)
	return plant.Status.State != apiv1.StateReady || uerr != nil, execErr
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
