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
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
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
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=get;list;watch
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete

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
	ctx = context.WithValue(ctx, "request", req.NamespacedName)
	logger := log.FromContext(ctx)
	logger.Info("Reconciling")

	// Fetch plant resource if it exists
	plant := &apiv1.Plant{}
	if err := r.Get(ctx, req.NamespacedName, plant); err != nil {
		logger.Info("Deleted successfully!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizers to plant if missing
	if controllerutil.AddFinalizer(plant, apiv1.Finalizer) {
		if err := r.Update(ctx, plant); err != nil { // TODO: this will reschedule
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant after finalizer check: %w", err))
		}
	}

	// Check if plant scheduled but not configured for deletion
	if !plant.DeletionTimestamp.IsZero() && plant.Status.State != apiv1.StateDeleting {
		if err := r.UpdateState(ctx, plant, apiv1.StateDeleting); err != nil {
			return r.ErrorHandle(ctx, plant, fmt.Errorf("could not update Plant status after triggering deletion: %w", err))
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle states
	return r.StateHandle(ctx, plant)
}

// ErrorHandle logs the error, puts plant into "Error" state, and requeue the request
func (r *PlantReconciler) ErrorHandle(ctx context.Context, plant *apiv1.Plant, err error) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, "error occurred, requeue...")
	return ctrl.Result{Requeue: true}, r.UpdateState(ctx, plant, apiv1.StateError)
}

// StateHandle invokes and reschedules workflows based on plant state.
func (r *PlantReconciler) StateHandle(ctx context.Context, plant *apiv1.Plant) (ctrl.Result, error) {
	switch plant.Status.State {
	case "": // change state to "Processing"
		log.FromContext(ctx).Info("Initial state with requeue...")
		return ctrl.Result{Requeue: true}, r.UpdateState(ctx, plant, apiv1.StateProcessing)

	case apiv1.StateProcessing, apiv1.StateError: // process until the state changes
		log.FromContext(ctx).Info("Processing state with requeue...")
		return ctrl.Result{Requeue: true}, r.HandleProcessingState(ctx, plant)

	case apiv1.StateDeleting: // try deletion until
		stillDeleting, err := r.HandleDeletingState(ctx, plant)
		log.FromContext(ctx).Info(fmt.Sprintf("Deleting state with requeue = %v...", stillDeleting || err != nil))
		return ctrl.Result{Requeue: stillDeleting}, err

	case apiv1.StateReady: // validate ready state
		log.FromContext(ctx).Info("Ready state running Processing one more time...")
		return ctrl.Result{}, r.HandleProcessingState(ctx, plant)
	}
	return ctrl.Result{}, nil
}

func (r *PlantReconciler) HandleProcessingState(ctx context.Context, plant *apiv1.Plant) error {
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
	}
	updateErr := r.UpdateState(ctx, plant, newState)
	if updateErr != nil {
		updateErr = fmt.Errorf("error while updating Plant status during processing: %w", updateErr)
	}
	return errors.Join(processingErr, updateErr)
}

func (r *PlantReconciler) HandleDeletingState(ctx context.Context, plant *apiv1.Plant) (bool, error) {
	// Remove hanging resources
	// The garbage collector will handle internal resource deletion, but we do the external ones

	// Nothing to do here for now, continue

	// Remove finalizers to notify that it is safe to delete
	controllerutil.RemoveFinalizer(plant, apiv1.Finalizer)
	if err := r.Update(ctx, plant); err != nil {
		return false, fmt.Errorf("error while trying to update Plant during deletion: %w", err)
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
	*plant.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if err := r.Client.Status().Update(ctx, plant); err != nil {
		return fmt.Errorf("could not update Plant status: %w", err)
	}
	return nil
}

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
