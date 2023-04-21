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
			required.DeepCopyInto(object) // update
			if err := controllerutil.SetControllerReference(plant, object, r.Client.Scheme()); err != nil {
				return err
			}
			return r.Client.Create(ctx, object)
		},
		UpdateFunc: func(object *corev1.Service) (bool, error) {
			if !reflect.DeepEqual(object.Spec, required.Spec) {
				object.Spec = required.Spec
				object.ObjectMeta.SetLabels(required.ObjectMeta.Labels)
				return true, r.Client.Update(ctx, object)
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels:    plant.GetLabels(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       *plant.Spec.ContainerPort,
					TargetPort: intstr.FromInt(int(*plant.Spec.ContainerPort)),
				},
			},
			Selector: plant.GetLabels(),
			Type:     corev1.ServiceTypeNodePort,
		},
	}
}
