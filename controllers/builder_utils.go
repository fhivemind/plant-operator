package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// notifyEvent will just inform who triggered the reconcile, usually used for resources tracking
func notifyWrapper(recorder record.EventRecorder, wrap predicate.Funcs) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal,
				"Sync",
				"Received create event on resource %T, reconciling", e.Object,
			)
			return wrap.Create(e)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal,
				"Sync",
				"Received delete event on resource %T, reconciling", e.Object,
			)
			return wrap.Delete(e)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			recorder.Eventf(e.ObjectNew, corev1.EventTypeNormal,
				"Sync",
				"Received update event on resource %T, reconciling", e.ObjectNew,
			)
			return wrap.Update(e)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal,
				"Sync",
				"Received generic event on resource %T, reconciling", e.Object,
			)
			return wrap.Generic(e)
		},
	}
}
