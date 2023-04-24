package workflow

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	"github.com/fhivemind/plant-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// newDeploymentHandler creates deployment resource.Executor for the given Plant
func (m *manager) newDeploymentHandler(plant *apiv1.Plant) resource.Executor[*appsv1.Deployment] {
	// Create expected object
	expected := defineDeployment(plant)
	m.Client().Scheme().Default(expected)

	// Return handler
	return resource.Executor[*appsv1.Deployment]{
		Name: "Deployment",
		FetchFunc: func(ctx context.Context, object *appsv1.Deployment) error {
			return m.Client().Get(ctx, types.NamespacedName{Namespace: expected.Namespace, Name: expected.Name}, object)
		},
		CreateFunc: func(ctx context.Context, object *appsv1.Deployment) error {
			expected.DeepCopyInto(object) // fill with required values
			if err := controllerutil.SetControllerReference(plant, object, m.Client().Scheme()); err != nil {
				return err
			}
			return m.Client().Create(ctx, object)
		},
		UpdateFunc: func(ctx context.Context, object *appsv1.Deployment) (bool, error) {
			diff := utils.Diff(&expected.Spec, &object.Spec)
			if diff.NotEqual() {
				expected.Spec.DeepCopyInto(&object.Spec)
				utils.MergeMapsSrcDst(expected.Labels, object.Labels)
				return true, m.Client().Update(ctx, object)
			}
			return false, diff.Error()
		},
		IsReady: func(_ context.Context, object *appsv1.Deployment) bool {
			return object.Status.AvailableReplicas == *plant.Spec.Replicas
		},
	}
}

func defineDeployment(plant *apiv1.Plant) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels:    plant.OperatorLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: plant.OperatorLabels(),
			},
			Replicas: plant.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: plant.OperatorLabels(),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            plant.Name,
							Image:           plant.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: *plant.Spec.ContainerPort,
								},
							},
						},
					},
				},
			},
		},
	}
}
