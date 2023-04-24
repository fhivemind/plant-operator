package controllers

import (
	"context"
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// UpdateStatus will update plants status using provided values and options.
func (r *PlantReconciler) UpdateStatus(ctx context.Context, plant *apiv1.Plant, opts ...func(*apiv1.Plant)) error {
	// update and send
	for _, opt := range opts {
		opt(plant)
	}
	if plant.Status.LastUpdateTime == nil {
		plant.Status.LastUpdateTime = new(metav1.Time)
	}
	*plant.Status.LastUpdateTime = metav1.Now()
	if err := r.Client.Status().Update(ctx, plant); err != nil {
		return fmt.Errorf("could not update Plant status: %w", err)
	}
	return nil
}

// UpdateResults will handle results from executions by adding them to Plant status
func (r *PlantReconciler) UpdateResults(ctx context.Context, plant *apiv1.Plant, results []resource.ExecuteResult) error {
	plant.Status.Resources = make([]apiv1.ResourceStatus, 0)

	// Handle child resources
	for _, res := range results {
		name := res.Name()
		obj := res.Object()

		// Update resource available condition
		ready := false
		reason := "WaitingForReadyState"
		message := fmt.Sprintf("Resource %T is in Not Ready state", obj)

		switch {
		case res.Error() != nil: // ERROR STATE
			message = fmt.Sprintf("Resource %T is in Error state", obj)

			r.Recorder.Eventf(plant, v1.EventTypeWarning,
				fmt.Sprintf("%sProcessing", name),
				"Reprocessing, Executed with error: %v", res.Error())

		case res.Skipped(): // SKIPPED STATE
			ready = true
			reason = "RefreshSkipped"
			message = fmt.Sprintf("Skipping resource %T as it was not requested", obj)

			r.Recorder.Eventf(plant, v1.EventTypeNormal,
				fmt.Sprintf("%sProcessing", name),
				"Change detected, %s", message)

		case res.Ready(): // READY STATE
			ready = true
			reason = "InReadyState"
			message = fmt.Sprintf("Resource %T is in Ready state", obj)

			if ops := res.ProcessingOps(); len(ops) > 0 {
				r.Recorder.Eventf(plant, v1.EventTypeNormal,
					fmt.Sprintf("%sProcessing", name),
					"Done, %s. Executed %s operation(s)", message, strings.Join(ops, ", "))
			} else {
				r.Recorder.Eventf(plant, v1.EventTypeNormal,
					fmt.Sprintf("%sProcessing", name),
					"Done, %s", message)
			}

		default: // PROCESSING STATE
			r.Recorder.Eventf(plant, v1.EventTypeNormal,
				fmt.Sprintf("%sProcessing", name),
				"Reprocessing, %s", message)
		}

		plant.UpdateCondition(apiv1.ConditionTypeAvailableFor(name), ready, reason, message)

		// Update plant resource status
		state := apiv1.StateProcessing
		if res.Error() != nil {
			state = apiv1.StateError
		} else if res.Ready() {
			state = apiv1.StateReady
		}

		if !res.Skipped() || obj != nil { // only add non-ignored and non-nil results
			plant.Status.Resources = append(plant.Status.Resources, apiv1.ResourceStatus{
				Name:  name,
				GVK:   obj.GetObjectKind().GroupVersionKind().String(),
				UID:   obj.GetUID(),
				State: state,
			})
		}
	}

	// Handle main state
	newState := plant.DetermineState()
	if newState == apiv1.StateReady && plant.Status.State != apiv1.StateReady {
		r.Recorder.Event(plant, v1.EventTypeNormal, "Ready", "All done, Plant is in Ready state")
	} else {
		eventType := v1.EventTypeNormal
		if newState == apiv1.StateError {
			eventType = v1.EventTypeWarning
		}
		r.Recorder.Eventf(plant, eventType, "Processing",
			"Reprocessing, Plant is in %s state due to conditions: %s",
			newState, strings.Join(plant.GetWaitingConditions(), ", "))
	}

	// return updated
	return r.UpdateStatus(ctx, plant, withState(newState))
}

func withState(state apiv1.State) func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.State = state
	}
}
