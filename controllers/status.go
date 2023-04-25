package controllers

import (
	"context"
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
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
		resObj := res.Object()
		resType := utils.ObjectType(resObj)

		// Get resource status
		ready := false
		reason := "WaitingReadyState"
		state := apiv1.StateProcessing
		message := fmt.Sprintf("Resource %s is in Not Ready state", resType)

		switch {
		case res.Errored(): // ERROR STATE
			state = apiv1.StateError
			message = fmt.Sprintf("Resource %s is in Error state", resType)

			r.Recorder.Eventf(plant, v1.EventTypeWarning, "Error", "Rescheduling as %s: %v", message, res.Error())
			break

		case res.Skipped(): // SKIPPED STATE
			ready = true
			reason = "ProcessingSkipped"
			state = apiv1.StateReady
			message = fmt.Sprintf("Resource %s skipped due to conditions", resType)

		case res.Ready(): // READY STATE
			ready = true
			reason = "InReadyState"
			state = apiv1.StateReady
			message = fmt.Sprintf("Resource %s is in Ready state", resType)
			if ops := res.ProcessingOps(); len(ops) > 0 {
				message = fmt.Sprintf("%s after %s ops", message, strings.Join(ops, ", "))
			}
		}

		// Update plant conditions and resources
		plant.UpdateCondition(apiv1.ConditionTypeAvailableFor(res.Name()), ready, reason, message)
		if !res.Skipped() || resObj != nil { // only add non-ignored and non-nil results
			plant.Status.Resources = append(plant.Status.Resources, apiv1.ResourceStatus{
				Name:  res.Name(),
				GVK:   resObj.GetObjectKind().GroupVersionKind().String(),
				UID:   resObj.GetUID(),
				State: state,
			})
		}
	}

	// Update plant main state
	newState := plant.DetermineState()
	switch newState {
	case apiv1.StateReady: // READY STATE
		r.Recorder.Event(plant, v1.EventTypeNormal, "Ready", "All tasks done, Plant is in Ready state")

	default: // ANY OTHER STATE FORCES RSYNC
		eventType := v1.EventTypeNormal
		if newState == apiv1.StateError {
			eventType = v1.EventTypeWarning
		}
		notReadyConds := strings.Join(plant.GetWaitingConditions(), ", ")
		r.Recorder.Eventf(plant, eventType, "WaitingReadyState", "Plant is in %s state due to conditions: %s", newState, notReadyConds)
	}

	// Return by updating
	return r.UpdateStatus(ctx, plant, withState(newState))
}

func withState(state apiv1.State) func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.State = state
	}
}
