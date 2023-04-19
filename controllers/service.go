package controllers

import (
	"context"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const serviceCondition v1.ConditionType = "deployment-service"

func (r *PlantReconciler) manageService(ctx context.Context, plant *v1.Plant) (*corev1.Service, error) {
	logger := log.FromContext(ctx)

	// Create service if not found
	service := &corev1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: plant.Name, Namespace: plant.Namespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, service); err != nil { // create if not found
				return nil, err
			}
			if err := controllerutil.SetControllerReference(plant, service, r.Scheme); err != nil { // set ownership
				return nil, err
			}
			logger.Info("successfully created service")
		} else {
			logger.Info("failed to create service")
			return nil, err
		}
	}

	// Update service if required
	requiredService := defineService(plant)
	if !reflect.DeepEqual(requiredService.Spec, service.Spec) {
		service.ObjectMeta = requiredService.ObjectMeta
		err = r.Client.Update(ctx, service)
		if err != nil {
			return nil, err
		}
		logger.Info("successfully updated service")
	}

	// TODO: handle resource changes by using watchers to handle Plant status updates
	plant.UpdateCondition(serviceCondition, metav1.ConditionTrue)

	// Return back
	return service, nil
}

func defineService(plant *v1.Plant) *corev1.Service {
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
					Name:       "http-service-port", // TODO: use static key
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
