package client

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var InvalidObjectTypeErr = fmt.Errorf("invalid object type, does not implement client.Object")

type CtxClient[T client.Object] interface {
	client.Client
	CreateOrFetch(ctx context.Context, obj T, opts ...Opts) error
}

func For[T client.Object](client client.Client) CtxClient[T] {
	return &ctxClient[T]{
		Client: client,
	}
}

type ctxClient[T client.Object] struct {
	client.Client
}

func (c ctxClient[T]) CreateOrFetch(ctx context.Context, obj T, opts ...Opts) error {
	logger := log.FromContext(ctx)

	// Load options
	options := getOption(opts...)
	typeName := fmt.Sprintf("%T", obj)

	// Fetch if possible
	err := c.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err == nil {
		logger.Info("successfully fetched", "type", typeName)
		return nil
	} else if !errors.IsNotFound(err) {
		logger.Info("failed to fetch", "type", typeName)
		return err
	}

	// Create if possible
	if err := c.Client.Create(ctx, obj); err != nil { // create
		return err
	}
	if options.owner != nil {
		if err := controllerutil.SetControllerReference(options.owner, obj, c.Client.Scheme()); err != nil { // set ownership
			return err
		}
	}
	logger.Info("successfully created", "type", typeName)
	return nil
}

type Opts func(*options)

type options struct {
	owner client.Object
}

func getOption(opts ...Opts) *options {
	result := new(options)
	for _, opt := range opts {
		opt(result)
	}
	return result
}
