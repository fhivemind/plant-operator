package controllers

import (
	"context"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	"github.com/fhivemind/plant-operator/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const serviceCondition v1.ConditionType = "deployment-service"

func (r *PlantReconciler) manageService(ctx context.Context, plant *v1.Plant) (*corev1.Service, error) {
	logger := log.FromContext(ctx)

	// Handle create/fetch
	required := defineService(plant)
	fetched := required.DeepCopy()
	err := client.For[*corev1.Service](r.Client).CreateOrFetch(ctx, fetched)
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
	plant.UpdateCondition(serviceCondition, metav1.ConditionTrue)

	// Return back
	return fetched, nil
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
