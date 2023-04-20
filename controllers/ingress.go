package controllers

import (
	"context"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/client"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const ingressCondition v1.ConditionType = "deployment-ingress"

// TODO: this will not work for ACME challenge, fix it!
func (r *PlantReconciler) manageIngress(ctx context.Context, plant *v1.Plant) (*networkingv1.Ingress, error) {
	logger := log.FromContext(ctx)

	// Handle create/fetch
	required := defineIngress(plant)
	fetched := required.DeepCopy()
	err := client.For[*networkingv1.Ingress](r.Client).CreateOrFetch(ctx, fetched)
	if err != nil {
		return nil, err
	}

	// Handle update
	if !reflect.DeepEqual(fetched.Spec, required.Spec) {
		fetched.ObjectMeta = required.ObjectMeta
		err = r.Client.Update(ctx, fetched)
		if err != nil {
			return nil, err
		}
		logger.Info("successfully updated deployment")
	}

	// TODO: handle resource changes by using watchers to handle Plant status updates
	plant.UpdateCondition(ingressCondition, metav1.ConditionTrue)

	// Return back
	return fetched, nil
}

func defineIngress(plant *v1.Plant) *networkingv1.Ingress {
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
