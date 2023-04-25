package controllers

import (
	"fmt"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// notifyWrapper will just inform who triggered the reconcile, usually used for resource tracking
func notifyWrapper(recorder record.EventRecorder, wrap predicate.Predicate) predicate.Funcs {
	withMsg := func(should bool, eventType string, obj client.Object) bool {
		if should {
			recorder.Eventf(obj, corev1.EventTypeNormal, "Sync", "Triggered for %s", eventType)
		}
		return should
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return withMsg(wrap.Create(e), "Create event", e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return withMsg(wrap.Delete(e), "Delete event", e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// get diff between objects
			eventInfo := "update event"
			if _, ok := e.ObjectOld.(*apiv1.Plant); ok {
				mapDiff, err := utils.UnsafeMapDiff(&e.ObjectOld.(*apiv1.Plant).Spec, &e.ObjectNew.(*apiv1.Plant).Spec)
				different := mapDiff.Values(true, false) // get only different values
				if err == nil && len(different) > 0 {
					eventInfo = fmt.Sprintf("Update event with %s", different)
				}
			}
			return withMsg(wrap.Update(e), eventInfo, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return withMsg(wrap.Generic(e), "Generic event", e.Object)
		},
	}
}
