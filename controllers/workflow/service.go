package workflow

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// newServiceHandler creates service resource.Executor for the given Plant
func (m *manager) newServiceHandler(plant *apiv1.Plant) resource.Executor[*corev1.Service] {
	// Create expected object
	expected := defineService(plant)
	m.Client().Scheme().Default(expected)

	// Return handler
	return resource.Executor[*corev1.Service]{
		Name: "Service",
		FetchFunc: func(ctx context.Context, object *corev1.Service) error {
			return m.Client().Get(ctx, types.NamespacedName{Namespace: expected.Namespace, Name: expected.Name}, object)
		},
		CreateFunc: func(ctx context.Context, object *corev1.Service) error {
			expected.DeepCopyInto(object) // fill with required values
			if err := controllerutil.SetControllerReference(plant, object, m.Client().Scheme()); err != nil {
				return err
			}
			return m.Client().Create(ctx, object)
		},
		UpdateFunc: func(ctx context.Context, object *corev1.Service) (bool, error) {
			diff := utils.Diff(&expected.Spec, &object.Spec)
			if diff.NotEqual() {
				expected.Spec.DeepCopyInto(&object.Spec)
				utils.MergeMapsSrcDst(expected.Labels, object.Labels)
				return true, m.Client().Update(ctx, object)
			}
			return false, diff.Error()
		},
		IsReady: func(_ context.Context, object *corev1.Service) bool {
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
