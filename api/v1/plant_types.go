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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

	// ResourceAnnotations specifies custom annotations to add to various resources.
	// Use this field for resource customization.
	// +optional
	ResourceAnnotations *ResourceAnnotations `json:"annotations"`
}

// ResourceAnnotations defines annotations for various resources managed by Plant.
// This field can be used for additional resource customization.
type ResourceAnnotations struct {
	Deployment map[string]string `json:"deployment,omitempty"`
	Pod        map[string]string `json:"pod,omitempty"`
	Service    map[string]string `json:"service,omitempty"`
	Ingress    map[string]string `json:"ingress,omitempty"`
}

// PlantStatus defines the observed state of Plant
type PlantStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Plant is the Schema for the plants API
type Plant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlantSpec   `json:"spec,omitempty"`
	Status PlantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PlantList contains a list of Plant
type PlantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Plant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Plant{}, &PlantList{})
}
