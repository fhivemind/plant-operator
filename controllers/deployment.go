package controllers

import (
	"context"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PlantReconciler) deploymentHandler(ctx context.Context, plant *apiv1.Plant) resource.Handler[*appsv1.Deployment] {
	required := defineDeployment(plant)
	return resource.Handler[*appsv1.Deployment]{
		Name: "deployment",
		Ctx:  ctx,
		FetchFunc: func(object *appsv1.Deployment) error {
			return r.Client.Get(ctx, types.NamespacedName{Namespace: required.Namespace, Name: required.Name}, object)
		},
		CreateFunc: func(object *appsv1.Deployment) error {
			object = required.DeepCopy() // update
			if err := r.Client.Create(ctx, object); err != nil {
				return err
			}
			return controllerutil.SetControllerReference(plant, object, r.Client.Scheme())
		},
		UpdateFunc: func(object *appsv1.Deployment) (bool, error) {
			if !reflect.DeepEqual(object.Spec, required.Spec) {
				object.Spec = required.Spec
				object.ObjectMeta.SetLabels(required.ObjectMeta.Labels)
				err := r.Client.Update(ctx, object)
				return true, err
			}
			return false, nil
		},
		IsReady: func(object *appsv1.Deployment) bool {
			return object.Status.AvailableReplicas != *plant.Spec.Replicas
		},
	}
}

func defineDeployment(plant *apiv1.Plant) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      plant.Name,
			Namespace: plant.Namespace,
			Labels: map[string]string{ // TODO: fill better
				"app": plant.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{ // TODO: fill better
					"app": plant.Name,
				},
			},
			Replicas: plant.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: plant.Namespace,
					Labels: map[string]string{ // TODO: fill better
						"app": plant.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            plant.Name,
							Image:           plant.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{ // TODO: expose API to specify port
								{
									Name:          "http-port", // TODO: use static key
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
}
