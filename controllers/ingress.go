package controllers

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PlantReconciler) ingressManager(ctx context.Context, plant *apiv1.Plant) resource.Handler[*networkingv1.Ingress] {
	required := defineIngress(plant)
	return resource.Handler[*networkingv1.Ingress]{
		Name: "ingress",
		Ctx:  ctx,
		FetchFunc: func(object *networkingv1.Ingress) error {
			return r.Client.Get(ctx, types.NamespacedName{Namespace: required.Namespace, Name: required.Name}, object)
		},
		CreateFunc: func(object *networkingv1.Ingress) error {
			object = required.DeepCopy() // update
			if err := r.Client.Create(ctx, object); err != nil {
				return err
			}
			return controllerutil.SetControllerReference(plant, object, r.Client.Scheme())
		},
		UpdateFunc: func(object *networkingv1.Ingress) (bool, error) {
			if !reflect.DeepEqual(object.Spec, required.Spec) {
				object.Spec = required.Spec
				object.ObjectMeta.SetLabels(required.ObjectMeta.Labels)
				err := r.Client.Update(ctx, object)
				return true, err
			}
			return false, nil
		},
		IsReady: func(object *networkingv1.Ingress) bool {
			// TODO: when we add TLS, we can check it here
			return true
		},
	}
}

func defineIngress(plant *apiv1.Plant) *networkingv1.Ingress {
	//var tlsIngress []networkingv1.IngressTLS
	//if tlsSecretName != "" {
	//	tlsIngress = []networkingv1.IngressTLS{
	//		{
	//			Hosts: []string{
	//				plant.Spec.Host,
	//			},
	//			SecretName: tlsSecretName,
	//		},
	//	}
	//}
	pathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels: map[string]string{ // TODO: fill better
				"app": plant.Name,
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: plant.Spec.IngressClassName,
			// TLS:              tlsIngress,
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
										Service: &networkingv1.IngressServiceBackend{ // TODO: handle better
											Name: plant.Name,
											Port: networkingv1.ServiceBackendPort{
												Name: "http-port",
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
