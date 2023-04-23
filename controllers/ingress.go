package controllers

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PlantReconciler) ingressManager(ctx context.Context, plant *apiv1.Plant) resource.Handler[*networkingv1.Ingress] {
	// create expected object
	expected := defineIngress(plant)
	r.Scheme.Default(expected)

	// return handler
	return resource.Handler[*networkingv1.Ingress]{
		Name: "ingress",
		FetchFunc: func(object *networkingv1.Ingress) error {
			return r.Client.Get(ctx, types.NamespacedName{Namespace: expected.Namespace, Name: expected.Name}, object)
		},
		CreateFunc: func(object *networkingv1.Ingress) error {
			expected.DeepCopyInto(object) // fill with required values
			if err := controllerutil.SetControllerReference(plant, object, r.Client.Scheme()); err != nil {
				return err
			}
			return r.Client.Create(ctx, object)
		},
		UpdateFunc: func(object *networkingv1.Ingress) (bool, error) {
			diff := utils.Diff(&expected.Spec, &object.Spec)
			if diff.NotEqual() {
				expected.Spec.DeepCopyInto(&object.Spec)
				utils.MergeMapsSrcDst(expected.Labels, object.Labels)
				return true, r.Client.Update(ctx, object)
			}
			return false, diff.Error()
		},
		IsReady: func(object *networkingv1.Ingress) bool {
			// TODO: when we add TLS, we can check it here
			return true
		},
	}
}

func defineIngress(plant *apiv1.Plant) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels:    plant.OperatorLabels(),
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: plant.Spec.IngressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: plant.Spec.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
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
