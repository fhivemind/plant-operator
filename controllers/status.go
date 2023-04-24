package controllers

import (
	"context"
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// UpdateStatus will update plants status using provided values and options.
func (r *PlantReconciler) UpdateStatus(ctx context.Context, plant *apiv1.Plant, opts ...func(*apiv1.Plant)) error {
	// update and send
	applyStatusOpts(plant, opts...)
	if plant.Status.LastUpdateTime == nil {
		plant.Status.LastUpdateTime = new(metav1.Time)
	}
	*plant.Status.LastUpdateTime = metav1.Now()
	if err := r.Client.Status().Update(ctx, plant); err != nil {
		return fmt.Errorf("could not update Plant status: %w", err)
	}
	return nil
}

func applyStatusOpts(plant *apiv1.Plant, opts ...func(*apiv1.Plant)) {
	for _, opt := range opts {
		opt(plant)
	}
}

func withState(state apiv1.State) func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.State = state
	}
}

func withClearedStatus() func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.Resources = make([]apiv1.ResourceStatus, 0)
	}
}

func withResults(results []resource.ExecuteResult) func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		for _, res := range results {
			// update Processed condition
			if ops := res.ProcessingOps(); len(ops) > 0 {
				plant.UpdateCondition(
					apiv1.ConditionTypesProcessedFor(res.Name()),
					true,
					fmt.Sprintf("%sDone", strings.Join(ops, "And")),
					fmt.Sprintf("Performed %s operations for %T", strings.Join(ops, ", "), res.Object()),
				)
			} else if res.Skipped() {
				plant.UpdateCondition(
					apiv1.ConditionTypesProcessedFor(res.Name()),
					true,
					"RemovedFromStack",
					fmt.Sprintf("Removed %T from watched resources", res.Object()),
				)
			}

			// update Available condition
			status := true
			reason := "InReadyState"
			msg := fmt.Sprintf("Object %T is in Ready state", res.Object())
			if err := res.Error(); err != nil {
				status = false
				reason = "WaitingForNonErrorState"
				msg = fmt.Sprintf("Object %T is in Error state, reason: %s", res.Object(), err.Error())
			} else if res.NotReady() {
				status = false
				reason = "WaitingForReadyState"
				msg = fmt.Sprintf("Object %T is in Not Ready state", res.Object())
			}

			if res.Skipped() {
				plant.UpdateCondition(
					apiv1.ConditionTypeAvailableFor(res.Name()),
					true,
					"RemovedFromStack",
					fmt.Sprintf("Removed %T from watched resources", res.Object()),
				)
			} else {
				plant.UpdateCondition(apiv1.ConditionTypeAvailableFor(res.Name()), status, reason, msg)
			}

			// Update resource
			state := apiv1.StateProcessing
			if res.Error() != nil {
				state = apiv1.StateError
			} else if res.Ready() {
				state = apiv1.StateReady
			}

			if !res.Skipped() || res.Object() == nil { // skip adding for ignored resources
				plant.Status.Resources = append(plant.Status.Resources, apiv1.ResourceStatus{
					Name:  res.Name(),
					GVK:   res.Object().GetObjectKind().GroupVersionKind().String(),
					UID:   res.Object().GetUID(),
					State: state,
				})
			}
		}
	}
}
