package controllers

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// logPredicate will just inform who triggered the reconcile, usually used for resources.
// TODO: maybe switch to eventing
func logPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			log.FromContext(context.Background()).
				Info(fmt.Sprintf("Received CREATE on subresource %T, reconciling", e.Object))
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.FromContext(context.Background()).
				Info(fmt.Sprintf("Received DELETE on subresource %T, reconciling", e.Object))
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.FromContext(context.Background()).
				Info(fmt.Sprintf("Received UPDATE on subresource %T, reconciling", e.ObjectNew))
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			log.FromContext(context.Background()).
				Info(fmt.Sprintf("Received GENERIC on subresource %T, reconciling", e.Object))
			return true
		},
	}
}
