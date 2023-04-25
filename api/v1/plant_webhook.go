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
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var plantlog = logf.Log.WithName("plant-resource")

func (r *Plant) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-operator-fhivemind-io-v1-plant,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.fhivemind.io,resources=plants,verbs=create;update,versions=v1,name=mplant.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Plant{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Plant) Default() {
	plantlog.Info("default", "name", r.Name)

	// set default ContainerPort
	if r.Spec.ContainerPort == nil {
		r.Spec.ContainerPort = new(int32)
		*r.Spec.ContainerPort = DefaultContainerPort
	}

	// set default Replicas
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = new(int32)
		*r.Spec.Replicas = DefaultReplicaCount
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-operator-fhivemind-io-v1-plant,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.fhivemind.io,resources=plants,verbs=create;update,versions=v1,name=vplant.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Plant{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Plant) ValidateCreate() error {
	plantlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Plant) ValidateUpdate(old runtime.Object) error {
	plantlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Plant) ValidateDelete() error {
	plantlog.Info("validate delete", "name", r.Name)
	return r.validate()
}

// validate runs general validation on Plant
func (r *Plant) validate() error {
	switch {
	case r.Spec.Image == "":
		return errors.New(".spec.image is required")

	case r.Spec.Host == "":
		return errors.New(".spec.host is required")

	case r.Spec.IngressClassName != nil && *r.Spec.IngressClassName == "":
		return errors.New(".spec.ingressClassName provided but empty")

	case r.Spec.TlsSecretName != nil && *r.Spec.TlsSecretName == "":
		return errors.New(".spec.tlsSecretName provided but empty")

	case r.Spec.TlsCertIssuerRef != nil && r.Spec.TlsCertIssuerRef.Name == "":
		return errors.New(".spec.tlsCertIssuerRef.Name cannot be empty")

	case r.Spec.TlsSecretName != nil && r.Spec.TlsCertIssuerRef != nil:
		return errors.New("both .spec.tlsSecretName and .spec.tlsCertIssuerRef provided but only one required")
	}
	return nil
}
