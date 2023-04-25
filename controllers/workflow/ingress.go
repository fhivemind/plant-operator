package workflow

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// newIngressHandler creates ingress resource.Executor for the given Plant.
// It also requires an tlsSecretName which will be used to determine
// if IngressTLS should be added to Ingress.
// If nil provided, it will not use IngressTLS (insecure Ingress).
func (m *manager) newIngressHandler(plant *apiv1.Plant, tlsSecretName *string) resource.Executor[*networkingv1.Ingress] {
	// Create expected object
	expected := defineIngress(plant, tlsSecretName)
	m.Client().Scheme().Default(expected)

	// Return handler
	return resource.Executor[*networkingv1.Ingress]{
		Name: "Ingress",
		FetchFunc: func(ctx context.Context, object *networkingv1.Ingress) error {
			return m.Client().Get(ctx, types.NamespacedName{Namespace: expected.Namespace, Name: expected.Name}, object)
		},
		CreateFunc: func(ctx context.Context, object *networkingv1.Ingress) error {
			expected.DeepCopyInto(object) // fill with required values
			if err := controllerutil.SetControllerReference(plant, object, m.Client().Scheme()); err != nil {
				return err
			}
			return m.Client().Create(ctx, object)
		},
		UpdateFunc: func(ctx context.Context, object *networkingv1.Ingress) (bool, error) {
			structDiff := utils.Diff(&expected.Spec, &object.Spec)
			tlsChanged := !reflect.DeepEqual(expected.Spec.TLS, object.Spec.TLS)
			ingressClassChanged := !reflect.DeepEqual(expected.Spec.IngressClassName, object.Spec.IngressClassName)
			if structDiff.NotEqual() || tlsChanged || ingressClassChanged {
				expected.Spec.DeepCopyInto(&object.Spec)
				utils.MergeMapsSrcDst(expected.Labels, object.Labels)
				return true, m.Client().Update(ctx, object)
			}
			return false, structDiff.Error()
		},
		IsReady: func(_ context.Context, object *networkingv1.Ingress) bool {
			// TODO: can use ping here to check if valid
			return true
		},
	}
}

func defineIngress(plant *apiv1.Plant, tlsSecretName *string) *networkingv1.Ingress {
	// Define TLS specs if provided
	var ingressTls []networkingv1.IngressTLS
	if tlsSecretName != nil {
		ingressTls = append(ingressTls, networkingv1.IngressTLS{
			Hosts:      []string{plant.Spec.Host},
			SecretName: *tlsSecretName,
		})
	}

	// Return Ingress
	ingressPathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels:    plant.OperatorLabels(),
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: plant.Spec.IngressClassName,
			TLS:              ingressTls,
			Rules: []networkingv1.IngressRule{
				{
					Host: plant.Spec.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &ingressPathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: plant.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: *plant.Spec.ContainerPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
