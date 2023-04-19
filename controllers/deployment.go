package controllers

import (
	"context"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const deploymentCondition v1.ConditionType = "deployment"

func (r *PlantReconciler) manageDeployment(ctx context.Context, plant *v1.Plant) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	// Create deployment if not found
	deployment := &appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: plant.Name, Namespace: plant.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, deployment); err != nil { // create if not found
				return nil, err
			}
			if err := controllerutil.SetControllerReference(plant, deployment, r.Scheme); err != nil { // set ownership
				return nil, err
			}
			logger.Info("successfully created deployment")
		} else {
			logger.Info("failed to create deployment")
			return nil, err
		}
	}

	// Update deployment if required
	requiredDeployment := defineDeployment(plant)
	if !reflect.DeepEqual(requiredDeployment.Spec, deployment.Spec) {
		deployment.ObjectMeta = requiredDeployment.ObjectMeta
		err = r.Client.Update(ctx, deployment)
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
	//err = r.SyncStatusObject(ctx, plant, &v1.ObjectStatus{
	//	UUID:  deployment.UID,
	//	State: deploymentState,
	//})
	plant.UpdateCondition(deploymentCondition, metav1.ConditionTrue)

	// Return back
	return deployment, nil
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
