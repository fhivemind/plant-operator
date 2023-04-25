package controllers_test

import (
	apiv1 "github.com/fhivemind/plant-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Default plant with image and host", Ordered, func() {
	plant := NewTestPlant("basic-plant")

	It("Should result in a ready state", func() {
		By("having transitioned the CR State to Ready after processing")
		Eventually(IsPlantInState(Ctx, plant.GetName(), apiv1.StateReady), Timeout, Interval).Should(BeTrue())
	})
})
