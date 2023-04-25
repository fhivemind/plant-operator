package controllers_test

import (
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plant with minimal configuration", Ordered, func() {
	plant := NewTestPlant("default-plant")
	RegisterPlant(plant)

	It("Should result in a valid state for image and host", func() {
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})
})

var _ = Describe("Plant with advanced configuration", Ordered, func() {
	plant := NewTestPlant("advanced-plant")
	RegisterPlant(plant)

	It("Should result in a valid state for custom container port", func() {
		plant.Spec.ContainerPort = new(int32)
		*plant.Spec.ContainerPort = 8080

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})

	It("Should result in a valid state for custom replica count", func() {
		plant.Spec.Replicas = new(int32)
		*plant.Spec.Replicas = 5

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})

	It("Should result in a valid state for custom ingress class", func() {
		plant.Spec.IngressClassName = new(string)
		*plant.Spec.IngressClassName = "nginx"

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})

	It("Should result in a valid state when custom config removed", func() {
		plant.Spec.ContainerPort = nil
		plant.Spec.Replicas = nil
		plant.Spec.IngressClassName = nil

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})
})

var _ = Describe("Plant with TLS", Ordered, func() {
	plant := NewTestPlant("tls-plant")
	RegisterPlant(plant)

	It("Should result in a valid state for tls secret", func() {
		plant.Spec.TlsSecretName = new(string)
		*plant.Spec.TlsSecretName = "tls-secret-name"

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})

	It("Should prioritize TlsCertIssuer over TlsSecret", func() {
		plant.Spec.TlsCertIssuerRef = &cmmeta.ObjectReference{
			Name: "custom-issuer",
		}

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})

	It("Should result in a valid state when Tls config removed", func() {
		plant.Spec.TlsCertIssuerRef = nil
		plant.Spec.TlsSecretName = nil

		SyncPlant(plant)
		Eventually(UNIT_IsPlantValid(plant), Timeout, Interval).Should(BeTrue())
	})
})

////E2E tests (disabled since I do have a running k8s CI/CD flow)
//var _ = Describe("Default plant with image and host", Ordered, func() {
//	plant := NewTestPlant("basic-plant")
//	RegisterPlant(plant)
//
//	It("Should result in a ready state", func() {
//		By("having transitioned the CR State to Ready after processing")
//		Eventually(IsPlantInState(Ctx, plant.GetName(), apiv1.StateReady), Timeout, Interval).Should(BeTrue())
//	})
//})
