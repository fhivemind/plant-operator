package controllers_test

import (
	"context"
	"fmt"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTestPlant(name string) *apiv1.Plant {
	return &apiv1.Plant{
		TypeMeta: v1.TypeMeta{
			Kind:       apiv1.PlantKind,
			APIVersion: apiv1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", name, randString(8)),
			Namespace: v1.NamespaceDefault,
		},
		Spec: apiv1.PlantSpec{
			Image: "dockerbogo/docker-nginx-hello-world:latest",
			Host:  "example.host",
		},
	}
}

func IsPlantInState(ctx context.Context, name string, state apiv1.State) func() bool {
	return func() bool {
		plant, err := GetPlant(ctx, name, "")
		if err != nil || plant.Status.State != state {
			return false
		}
		return true
	}
}

func SyncPlant(plant *apiv1.Plant) error {
	return PlantClient.Get(Ctx, client.ObjectKey{
		Name:      plant.Name,
		Namespace: plant.Namespace,
	}, plant)
}

func GetPlant(ctx context.Context, name, namespace string) (*apiv1.Plant, error) {
	plant := &apiv1.Plant{}
	if namespace == "" {
		namespace = v1.NamespaceDefault
	}
	if err := PlantClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, plant); err != nil {
		return nil, err
	}
	return plant, nil
}

func GetDeployment(p *apiv1.Plant) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	if err := PlantClient.Get(Ctx, client.ObjectKey{Name: p.Name, Namespace: p.Namespace}, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func GetService(p *apiv1.Plant) (*corev1.Service, error) {
	service := &corev1.Service{}
	if err := PlantClient.Get(Ctx, client.ObjectKey{Name: p.Name, Namespace: p.Namespace}, service); err != nil {
		return nil, err
	}
	return service, nil
}

func GetIngress(p *apiv1.Plant) (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{}
	if err := PlantClient.Get(Ctx, client.ObjectKey{Name: p.Name, Namespace: p.Namespace}, ingress); err != nil {
		return nil, err
	}
	return ingress, nil
}

func GetCertificate(p *apiv1.Plant) (*cmv1.Certificate, error) {
	cert := &cmv1.Certificate{}
	if err := PlantClient.Get(Ctx, client.ObjectKey{Name: p.Name, Namespace: p.Namespace}, cert); err != nil {
		return nil, err
	}
	return cert, nil
}
