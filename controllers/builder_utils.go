package controllers

import (
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
			recorder.Eventf(obj, corev1.EventTypeNormal,
				"Sync",
				"Triggered for %s event on resource %s", eventType, utils.ObjectType(obj),
			)
		}
		return should
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return withMsg(wrap.Create(e), "create", e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return withMsg(wrap.Delete(e), "delete", e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return withMsg(wrap.Update(e), "update", e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return withMsg(wrap.Generic(e), "generic", e.Object)
		},
	}
}
