package controllers

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const serviceCondition apiv1.ConditionType = "deployment-service"

func (r *PlantReconciler) serviceManager(ctx context.Context, plant *apiv1.Plant) resource.Handler[*corev1.Service] {
	required := defineService(plant)
	return resource.Handler[*corev1.Service]{
		Name: "service",

		FetchFunc: func(object *corev1.Service) error {
			return r.Client.Get(ctx, types.NamespacedName{Namespace: required.Namespace, Name: required.Name}, object)
		},

		CreateFunc: func(object *corev1.Service) error {
			if err := r.Client.Create(ctx, object); err != nil {
				return err
			}
			return controllerutil.SetControllerReference(plant, object, r.Client.Scheme())
		},

		UpdateFunc: func(object *corev1.Service) (bool, error) {
			if !reflect.DeepEqual(object.Spec, required.Spec) {
				object.ObjectMeta = required.ObjectMeta
				err := r.Client.Update(ctx, object)
				return true, err
			}
			return false, nil
		},

		IsReady: func(object *corev1.Service) bool {
			return apiv1.ConditionsReady(object.Status.Conditions)
		},
	}
}

func defineService(plant *apiv1.Plant) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels: map[string]string{ // TODO: fill better
				"app": plant.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http-port", // TODO: use static key
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromString("http-port"), // TODO: use static key
				},
			},
			Selector: map[string]string{
				"app": plant.Name, // TODO: fill better
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
}
