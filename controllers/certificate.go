package controllers

import (
	"fmt"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/fhivemind/plant-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
