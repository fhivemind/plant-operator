# Plant Operator v1

A Kubernetes operator that deploys and manages an application using custom deployment and networking rules.
It is intended to deploy and expose images on-the-fly best suited for simple demos and testing environments.
_Any changes to the underlying resources managed by Plant operator will force reconciliation to values
specified by the CRD._

The specifications of the operator are divided into two sections which control individual workflows.

### Deployment

- `image` (required): specifies the image to use for Deployment containers.
- `containerPort` (optional, defaults to 80): the container port to expose for host traffic.
- `replicas` (optional, defaults to 1): the number of desired pods to deploy.

### Networking

- `host` (required): the domain name of a network host where the deployed image will be accessible.
- `ingressClassName` (optional): the name of the Ingress controller to use.
- `tlsSecretName` (optional): the name of an existing TLS secret for the given host.
- `certIssuerRef` (optional): the name of local Issuer to use for obtaining certificates.

Note: If both `tlsSecretName` and `certIssuerRef` are specified, `tlsSecretName` will be prioritized.

## Examples

```yaml
apiVersion: operator.cisco.io/v1
kind: Plant
metadata:
  name: example
spec:
  image: nginx:latest
  containerPort: 8080
  replicas: 3
  host: example.com
  ingressClassName: nginx
  certIssuerRef:
    name: my-issuer
```