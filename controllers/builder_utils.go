package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// notifyEvent will just inform who triggered the reconcile, usually used for resources tracking
func notifyEvent(recorder record.EventRecorder) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal, "Trigger",
				"Received CREATE event on resource %T, reconciling", e.Object)
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal, "Trigger",
				"Received DELETE event on resource %T, reconciling", e.Object)
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			recorder.Eventf(e.ObjectNew, corev1.EventTypeNormal, "Trigger",
				"Received UPDATE event on resource %T, reconciling", e.ObjectNew)
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			recorder.Eventf(e.Object, corev1.EventTypeNormal, "Trigger",
				"Received UPDATE event on resource %T, reconciling", e.Object)
			return true
		},
	}
}
