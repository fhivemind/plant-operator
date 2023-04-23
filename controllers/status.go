package controllers

import (
	"context"
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateStatus will update plants status using provided values and options.
func (r *PlantReconciler) UpdateStatus(ctx context.Context, plant *apiv1.Plant, opts ...func(*apiv1.Plant)) error {
	// update status from opts
	before := plant.Status.State
	for _, opt := range opts {
		opt(plant)
	}
	after := plant.Status.State

	// update and send
	if plant.Status.LastUpdateTime == nil {
		plant.Status.LastUpdateTime = new(metav1.Time)
	}
	*plant.Status.LastUpdateTime = metav1.Now()
	if err := r.Client.Status().Update(ctx, plant); err != nil {
		return fmt.Errorf("could not update Plant status (%s => %s): %w", before, after, err)
	}
	return nil
}

func withState(state apiv1.State) func(*apiv1.Plant) {
	return func(plant *apiv1.Plant) {
		plant.Status.State = state
	}
}
