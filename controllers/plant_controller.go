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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1 "github.com/fhivemind/plant-operator/api/v1"
)

// SetupWithManager sets up the controller with the Manager.
func (r *PlantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Plant{}).
		Complete(r)
}

// PlantReconciler reconciles a Plant object
type PlantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.cisco.io,resources=plants/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking,resources=ingress,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Plant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PlantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("plant", req.NamespacedName)
	logger.Info("Reconciling")

	// Fetch plant resource if it exists
	plant := &apiv1.Plant{}
	if err := r.Get(ctx, req.NamespacedName, plant); err != nil {
		logger.Info("Deleted successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizers to plant if missing
	if !controllerutil.ContainsFinalizer(plant, apiv1.PlantFinalizer) {
		controllerutil.AddFinalizer(plant, apiv1.PlantFinalizer)
		if err := r.Update(ctx, plant); err != nil {
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant after finalizer check: %w", err))
		}
	}

	// Check if plant scheduled but not configured for deletion
	if !plant.DeletionTimestamp.IsZero() && plant.Status.State != apiv1.StateDeleting {
		if err := r.UpdateStatusState(ctx, plant, apiv1.StateDeleting); err != nil {
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant status after triggering deletion: %w", err))
		}
	}

	// Handle states
	return r.StateHandle(ctx, plant)
}

// ErrorHandle logs the error, puts plant into "Error" state, and requeue the request
func (r *PlantReconciler) ErrorHandle(ctx context.Context, plant *apiv1.Plant, err error) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, "error occurred, requeue...")
	return ctrl.Result{Requeue: true}, r.UpdateStatusState(ctx, plant, apiv1.StateError)
}

// StateHandle invokes and reschedules workflows based on plant state.
func (r *PlantReconciler) StateHandle(ctx context.Context, plant *apiv1.Plant) (ctrl.Result, error) {
	switch plant.Status.State {
	case "": // change state to "Processing"
		return ctrl.Result{}, r.UpdateStatusState(ctx, plant, apiv1.StateProcessing)

	case apiv1.StateProcessing, apiv1.StateError: // process until the state changes
		return ctrl.Result{Requeue: true}, r.HandleProcessingState(ctx, plant)

	case apiv1.StateDeleting: // try deletion until
		stillDeleting, err := r.HandleDeletingState(ctx, plant)
		return ctrl.Result{Requeue: stillDeleting}, err

	case apiv1.StateReady: // validate ready state
		return ctrl.Result{}, r.HandleProcessingState(ctx, plant)
	}
	return ctrl.Result{}, nil
}

func (r *PlantReconciler) HandleProcessingState(ctx context.Context, plant *apiv1.Plant) error {
	// create one by one item

	return nil
}

func (r *PlantReconciler) HandleDeletingState(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	// remove resources
	// if removing { return true, err }

	// remove finalizers to notify that it is safe to delete
	controllerutil.RemoveFinalizer(plant, apiv1.PlantFinalizer)
	if err := r.Update(ctx, plant); err != nil {
		return false, fmt.Errorf("error while trying to update plant during deletion: %w", err)
	}
	return false, nil
}

func (r *PlantReconciler) UpdateStatusState(ctx context.Context, plant *apiv1.Plant, newState apiv1.State) error {
	plant.Status.State = newState
	return r.UpdateStatus(ctx, plant)
}

func (r *PlantReconciler) UpdateStatus(ctx context.Context, plant *apiv1.Plant) error {
	plant.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if err := r.Patch(ctx, plant, client.Apply); err != nil {
		return fmt.Errorf("could not update Plant status: %w", err)
	}
	return nil
}

func (r *PlantReconciler) SyncStatusObject(ctx context.Context, plant *apiv1.Plant, object *apiv1.ObjectStatus) error {
	found := false
	for id, obj := range plant.Status.Objects {
		if obj.UUID == object.UUID {
			found = true
			plant.Status.Objects[id] = *object.DeepCopy()
			break
		}
	}
	if !found {
		plant.Status.Objects = append(plant.Status.Objects, *object.DeepCopy())
	}
	return r.UpdateStatus(ctx, plant)
}
