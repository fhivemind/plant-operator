package controllers

import (
	"context"
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
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

func withClearedResources() func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.Resources = make([]apiv1.ResourceStatus, 0)
	}
}

// withResults will handle results from executions by adding them to Plant status
// TODO: very ugly code, maybe fix a bit
func withResults(eventRecorder record.EventRecorder, results []resource.ExecuteResult) func(*apiv1.Plant) {

	return func(plant *apiv1.Plant) {
		for _, res := range results {
			// get dependencies
			name := res.Name()
			obj := res.Object()
			ops := res.ProcessingOps()
			opsStr := strings.Join(ops, ", ")

			// update Processed condition
			if len(ops) > 0 {
				plant.UpdateCondition(
					apiv1.ConditionTypesProcessedFor(name),
					true,
					fmt.Sprintf("%sDone", strings.Join(ops, "And")),
					fmt.Sprintf("Performed %s operations for %T", opsStr, obj),
				)
			} else if res.Skipped() {
				plant.UpdateCondition(
					apiv1.ConditionTypesProcessedFor(name),
					true,
					"RemovedFromStack",
					fmt.Sprintf("Removed %T from watched resources", obj),
				)
			}

			// update Available condition
			status := true
			reason := "InReadyState"
			msg := fmt.Sprintf("Object %T is in Ready state", obj)
			if err := res.Error(); err != nil {
				status = false
				reason = "WaitingForNonErrorState"
				msg = fmt.Sprintf("Object %T is in Error state", obj)

				eventRecorder.Eventf(plant, v1.EventTypeWarning, fmt.Sprintf("%sProcessing", name),
					"Handling resource %T exited with error: %s", obj, err.Error())
			} else if res.NotReady() {
				status = false
				reason = "WaitingForReadyState"
				msg = fmt.Sprintf("Object %T is in Not Ready state", obj)

				eventRecorder.Eventf(plant, v1.EventTypeWarning, fmt.Sprintf("%sProcessing", name),
					"Resource %T is not yet in Ready state", obj)
			} else if len(ops) > 0 {
				eventRecorder.Eventf(plant, v1.EventTypeWarning, fmt.Sprintf("%sProcessing", name),
					"Successfully performed %s operation(s) for resource %T", opsStr, obj)
			}

			if res.Skipped() {
				plant.UpdateCondition(
					apiv1.ConditionTypeAvailableFor(name),
					true,
					"RemovedFromStack",
					fmt.Sprintf("Removed %T from watched resources", obj),
				)
			} else {
				plant.UpdateCondition(apiv1.ConditionTypeAvailableFor(name), status, reason, msg)
			}

			// Update resource
			state := apiv1.StateProcessing
			if res.Error() != nil {
				state = apiv1.StateError
			} else if res.Ready() {
				state = apiv1.StateReady
			}

			if !res.Skipped() || obj == nil { // skip adding for ignored resources
				plant.Status.Resources = append(plant.Status.Resources, apiv1.ResourceStatus{
					Name:  name,
					GVK:   obj.GetObjectKind().GroupVersionKind().String(),
					UID:   obj.GetUID(),
					State: state,
				})
			}
		}
	}
}
