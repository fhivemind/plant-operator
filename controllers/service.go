package controllers

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PlantReconciler) serviceManager(ctx context.Context, plant *apiv1.Plant) resource.Handler[*corev1.Service] {
	// create expected object
	expected := defineService(plant)
	r.Scheme.Default(expected)

	// return handler
	return resource.Handler[*corev1.Service]{
		Name: "service",
		FetchFunc: func(object *corev1.Service) error {
			return r.Client.Get(ctx, types.NamespacedName{Namespace: expected.Namespace, Name: expected.Name}, object)
		},
		CreateFunc: func(object *corev1.Service) error {
			expected.DeepCopyInto(object) // fill with required values
			if err := controllerutil.SetControllerReference(plant, object, r.Client.Scheme()); err != nil {
				return err
			}
			return r.Client.Create(ctx, object)
		},
		UpdateFunc: func(object *corev1.Service) (bool, error) {
			expectedSpecsMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&expected.Spec)
			if err != nil {
				return false, err
			}
			objectSpecsMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&object.Spec)
			if err != nil {
				return false, err
			}
			if !equality.Semantic.DeepDerivative(expectedSpecsMap, objectSpecsMap) {
				expected.Spec.DeepCopyInto(&object.Spec)
				utils.MergeMapsSrcDst(expected.Labels, object.Labels)
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
			Labels:    plant.OperatorLabels(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       *plant.Spec.ContainerPort,
					TargetPort: intstr.FromInt(int(*plant.Spec.ContainerPort)),
				},
			},
			Selector: plant.OperatorLabels(),
			Type:     corev1.ServiceTypeNodePort,
		},
	}
}
