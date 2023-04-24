package workflow

import (
	"context"
	"errors"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ClientNotConfiguredErr = errors.New("manager client not configured")

// Manager runs required workflow without modifying the Plant object.
// It abstracts the operator execution from configuration and simplifies
// the overall dependency management.
// TODO: Tests can use mocked Manager interface
type Manager interface {
	// Managed returns all Kubernetes objects managed by the Manager.
	// It is safe to call this method on non-initialized Manager.
	Managed() []client.Object

	// Execute executes the overall workflow. It will interact with Kubernetes API,
	// so make sure to use WithClient before execution.
	// If the client is not set, it returns ClientNotConfiguredErr error.
	Execute(ctx context.Context, plant *apiv1.Plant) ([]resource.ExecuteResult, error)

	// Client returns the current Manager Kubernetes client
	Client() client.Client

	// WithClient sets the current Manager Client. Must be called before Run.
	WithClient(client client.Client) Manager
}

// NewManager creates a bare Manager.
// Before executing Manager.Run, make sure to configure client via Manager.WithClient
func NewManager() Manager {
	return &manager{}
}

type manager struct {
	client client.Client
}

func (m *manager) Managed() []client.Object {
	return []client.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
		&networkingv1.Ingress{},
		&certv1.Certificate{},
	}
}

func (m *manager) Client() client.Client { return m.client }

func (m *manager) Execute(ctx context.Context, plant *apiv1.Plant) ([]resource.ExecuteResult, error) {
	// Check client
	if m.client == nil {
		return nil, ClientNotConfiguredErr
	}

	// Do processing for each handler
	procGroup := errgroup.Group{}
	results := make([]resource.ExecuteResult, 4)

	// Execute deployment
	deployment := &appsv1.Deployment{}
	service := &corev1.Service{}
	procGroup.Go(func() error { return runWith(ctx, deployment, m.newDeploymentHandler(plant), &results[0]) })
	procGroup.Go(func() error { return runWith(ctx, service, m.newServiceHandler(plant), &results[1]) })

	// Execute networking
	certificate := &certv1.Certificate{}
	ingress := &networkingv1.Ingress{}
	tlsSecretName, tlsHandler := m.newTlsOrNopHandler(plant)
	procGroup.Go(func() error { return runWith(ctx, certificate, tlsHandler, &results[2]) })
	procGroup.Go(func() error { return runWith(ctx, ingress, m.newIngressHandler(plant, tlsSecretName), &results[3]) })

	// Return
	return results, procGroup.Wait()
}

func (m *manager) WithClient(client client.Client) Manager {
	m.client = client
	return m
}

// runWith handles sub-resource execution using dynamic resource.Executor
func runWith[T client.Object](ctx context.Context, obj T, handler resource.Executor[T], result *resource.ExecuteResult) error {
	*result = handler.Execute(ctx, obj)
	return result.Error()
}
