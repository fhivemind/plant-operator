package controllers

import (
	"context"
	"fmt"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const certificateCondition v1.ConditionType = "deployment-certification"

func (r *PlantReconciler) manageCertificate(ctx context.Context, plant *v1.Plant) (string, error) {
	logger := log.FromContext(ctx)

	if plant.Spec.CertIssuerRef != nil {
		// Create certificate if not found
		certificate := &certv1.Certificate{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: plant.Name, Namespace: plant.Namespace}, certificate)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.Client.Create(ctx, certificate); err != nil { // create if not found
					return "", err
				}
				if err := controllerutil.SetControllerReference(plant, certificate, r.Scheme); err != nil { // set ownership
					return "", err
				}
				logger.Info("successfully created certificate")
			} else {
				logger.Info("failed to create certificate")
				return "", err
			}
		}

		// Update certificate if required
		requiredCertificate := defineCertificate(plant)
		if !reflect.DeepEqual(requiredCertificate.Spec, certificate.Spec) {
			certificate.ObjectMeta = requiredCertificate.ObjectMeta
			err = r.Client.Update(ctx, certificate)
			if err != nil {
				return "", err
			}
			logger.Info("successfully updated certificate")
		}
	}

	// TODO: handle resource changes by using watchers to handle Plant status updates
	plant.UpdateCondition(certificateCondition, metav1.ConditionTrue)

	// Return back
	return tlsSecretNameFor(plant), nil
}

func defineCertificate(plant *v1.Plant) *certv1.Certificate {
	return &certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsSecretNameFor(plant),
			Namespace: plant.Namespace,
		},
		Spec: certv1.CertificateSpec{
			SecretName: tlsSecretNameFor(plant),
			DNSNames: []string{
				plant.Spec.Host,
			},
			IssuerRef: *plant.Spec.CertIssuerRef,
		},
	}
}

// TODO: fix this function, very buggy
func tlsSecretNameFor(plant *v1.Plant) string {
	if plant.Spec.CertIssuerRef != nil {
		return fmt.Sprintf("%s-tls", plant.Name)
	} else if plant.Spec.TlsSecretRef != nil {
		return *plant.Spec.TlsSecretRef
	}
	return ""
}
