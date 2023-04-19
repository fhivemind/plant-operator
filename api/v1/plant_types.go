/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PlantSpec defines the desired state of Plant
type PlantSpec struct {
	// Image specifies the image use for Deployment containers
	//+kubebuilder:validation:Required
	Image string `json:"image,omitempty"`

	// Replicas defines the number of desired pods to deploy. Defaults to 1.
	//+kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Host defines the domain name of a network host where the deployed image will be accessible.
	// Follows RFC 3986 standard.
	//+kubebuilder:validation:Required
	Host string `json:"host,omitempty"`

	// TODO: Improve interface
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`
	// +optional
	TlsSecretRef *string `json:"tlsSecretRef,omitempty"`
	// +optional
	CertIssuerRef *cmmeta.ObjectReference `json:"issuerRef,omitempty"`
}

// PlantStatus defines the observed state of Plant
// +kubebuilder:subresource:status
type PlantStatus struct {
	// State signifies current state of Plant.
	State State `json:"state,omitempty"`

	// Conditions defines a list which indicates the status of the Plant.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Objects contains various identifiers about managed objects' states.
	Objects []ObjectStatus `json:"objects,omitempty"`

	// LastUpdateTime specifies the last time this resource has been updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// ObjectStatus defines the observed state of Plant-managed or other objects.
// If more context is required, embed into the object.
type ObjectStatus struct {
	UUID  types.UID `json:"uuid,omitempty"`
	State State     `json:"state,omitempty"`
}

// ConditionType sets the type to a concrete type for safety.
type ConditionType string

// State defines all possible resource states
// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error;""
type State string

const (
	// StateReady implies that the resource is ready and has been installed successfully.
	StateReady State = "Ready"
	// StateProcessing implies that the resource has just started or is being fixed by reconciliation.
	StateProcessing State = "Processing"
	// StateError implies an error for the resource occurred. The state can during next reconsiliation.
	StateError State = "Error"
	// StateDeleting implies the resource is being deleted.
	StateDeleting State = "Deleting"
)

// PlantList contains a list of Plant
// +kubebuilder:object:root=true
type PlantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Plant `json:"items"`
}

// Plant is the Schema for the plants API.
// +kubebuilder:object:root=true
type Plant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlantSpec   `json:"spec,omitempty"`
	Status PlantStatus `json:"status,omitempty"`
}

// DetermineState returns calculated state from objects and conditions.
func (plant *Plant) DetermineState() State {
	status := &plant.Status
	for _, moduleStatus := range status.Objects {
		if moduleStatus.State == StateError {
			return StateError
		}
	}
	for _, condition := range status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return StateProcessing
		}
	}
	return StateReady
}

// ConditionsReady returns true if all Conditions are satisfied for Plant.
func (plant *Plant) ConditionsReady() bool {
	if len(plant.Status.Conditions) == 0 {
		return false
	}
	for _, condition := range plant.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return false
		}
	}
	return true
}

// UpdateCondition updates specific condition based on type.
func (plant *Plant) UpdateCondition(conditionType ConditionType, status metav1.ConditionStatus) {
	reason := "not ready"
	msg := fmt.Sprintf("%s is not in ready state", conditionType)
	if status == metav1.ConditionTrue {
		reason = "ready"
		msg = fmt.Sprintf("%s is in ready state", conditionType)
	}

	meta.SetStatusCondition(&plant.Status.Conditions, metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: plant.GetGeneration(),
	})
}

// ContainsCondition returns true if the given condition is equal to any of the statuses.
func (plant *Plant) ContainsCondition(conditionType ConditionType, conditionStatus ...metav1.ConditionStatus) bool {
	for _, existingCondition := range plant.Status.Conditions {
		if existingCondition.Type != string(conditionType) {
			continue
		}
		if len(conditionStatus) > 0 {
			for i := range conditionStatus {
				if existingCondition.Status == conditionStatus[i] {
					return true
				}
			}
		} else {
			return true
		}
	}
	return false
}

func init() {
	SchemeBuilder.Register(&Plant{}, &PlantList{})
}
