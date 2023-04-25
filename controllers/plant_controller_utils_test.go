package controllers_test

import (
	"fmt"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
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

func RegisterPlant(plant *apiv1.Plant) {
	BeforeAll(func() {
		Expect(PlantClient.Create(Ctx, plant)).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		Expect(PlantClient.Delete(Ctx, plant)).NotTo(HaveOccurred())
	})
}

// SyncPlant updates Plant by changing fresh Plants specs to avoid updating old resources.
// Retries maximum of 4 times (or 1s)
func SyncPlant(plant *apiv1.Plant) {
	Eventually(func() bool {
		freshPlant, err := GetPlant(plant.Name, plant.Namespace)
		if err != nil {
			return false
		}
		plant.Spec.DeepCopyInto(&freshPlant.Spec)
		freshPlant.DeepCopyInto(plant)
		if err := PlantClient.Update(Ctx, plant); err != nil {
			return false
		}
		return true
	}, 1*time.Second, Interval).Should(BeTrue())
}

func GetPlant(name, namespace string) (*apiv1.Plant, error) {
	plant := &apiv1.Plant{}
	if namespace == "" {
		namespace = v1.NamespaceDefault
	}
	if err := PlantClient.Get(Ctx, client.ObjectKey{Namespace: namespace, Name: name}, plant); err != nil {
		return nil, err
	}
	return plant, nil
}

func GetPlantPort(plant *apiv1.Plant) int32 {
	if plant.Spec.ContainerPort == nil {
		return apiv1.DefaultContainerPort
	}
	return *plant.Spec.ContainerPort
}

func getPlantReplicas(plant *apiv1.Plant) int32 {
	if plant.Spec.Replicas == nil {
		return apiv1.DefaultReplicaCount
	}
	return *plant.Spec.Replicas
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

// for unit tests, it's a bit too much, but okay
func UNIT_IsPlantValid(plant *apiv1.Plant) func() bool {
	return func() bool {
		// Fetch data
		deployment, err := GetDeployment(plant)
		if err != nil {
			return false
		}

		service, err := GetService(plant)
		if err != nil {
			return false
		}

		cert, err := GetCertificate(plant)
		if plant.Spec.TlsCertIssuerRef != nil && err != nil {
			return false
		}

		ingress, err := GetIngress(plant)
		if err != nil {
			return false
		}

		// Check Deployment
		found := false
	mainLoop:
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Image == plant.Spec.Image {
				for _, port := range container.Ports {
					if GetPlantPort(plant) == port.ContainerPort {
						found = true
						break mainLoop
					}
				}
			}
		}
		if !found || *deployment.Spec.Replicas != getPlantReplicas(plant) {
			return false
		}

		// Check Service
		found = false
		for _, port := range service.Spec.Ports {
			requiredPort := GetPlantPort(plant)
			requiredTargetPort := intstr.FromInt(int(requiredPort))
			if requiredPort == port.Port && requiredTargetPort == port.TargetPort {
				found = true
				break
			}
		}
		if !found {
			return false
		}

		// Check Certs
		secretName := plant.Spec.TlsSecretName
		if cert != nil && plant.Spec.TlsCertIssuerRef != nil {
			secretName = &cert.Spec.SecretName

			found = false
			for _, dns := range cert.Spec.DNSNames {
				if dns == plant.Spec.Host {
					found = true
					break
				}
			}
			if !found || !reflect.DeepEqual(*plant.Spec.TlsCertIssuerRef, cert.Spec.IssuerRef) {
				return false
			}
		}

		// Fetch Ingress
		found = false
	mainIngressLoop:
		for _, rule := range ingress.Spec.Rules {
			if rule.Host == plant.Spec.Host {
				for _, path := range rule.HTTP.Paths {
					if path.Backend.Service.Port.Number == GetPlantPort(plant) {
						found = true
						break mainIngressLoop
					}
				}
			}
		}
		if !found || !reflect.DeepEqual(ingress.Spec.IngressClassName, plant.Spec.IngressClassName) {
			return false
		}
		if secretName != nil {
			found = false
		mainTlsLoop:
			for _, tls := range ingress.Spec.TLS {
				if tls.SecretName == *secretName {
					for _, host := range tls.Hosts {
						if host == plant.Spec.Host {
							found = true
							break mainTlsLoop
						}
					}
				}
			}
			if !found {
				return false
			}
		}

		return true
	}
}

// for E2E tests
func ETE_IsPlantInState(name string, state apiv1.State) func() bool {
	return func() bool {
		plant, err := GetPlant(name, "")
		if err != nil || plant.Status.State != state {
			return false
		}
		return true
	}
}

func randString(n int) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}
