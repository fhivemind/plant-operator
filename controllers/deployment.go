package controllers

import (
	"context"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const deploymentCondition v1.ConditionType = "deployment"

func (r *PlantReconciler) manageDeployment(ctx context.Context, plant *v1.Plant) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	// Handle create/fetch
	required := defineDeployment(plant)
	fetched := required.DeepCopy()
	err := client.For[*appsv1.Deployment](r.Client).CreateOrFetch(ctx, fetched)
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
	//// Verify deployment
	//deploymentState, deploymentStatus := v1.StateProcessing, metav1.ConditionFalse
	//if deployment.Status.ReadyReplicas == *plant.Spec.Replicas {
	//	deploymentState, deploymentStatus = v1.StateReady, metav1.ConditionTrue
	//}
	//err = r.SyncStatusObject(ctx, plant, &v1.ResourceStatus{
	//	UUID:  deployment.UID,
	//	State: deploymentState,
	//})
	plant.UpdateCondition(deploymentCondition, metav1.ConditionTrue)

	// Return back
	return fetched, nil
}

func defineDeployment(plant *v1.Plant) *appsv1.Deployment {
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
