# Plant Operator

Plant operator manages applications by simplifying deployments and networking rules.
It is intended to quickly and safely deploy containers in cloud-native fashion.

Under the hood, it relies on deployments, services, and ingress resources, while also allowing for custom flavours of 
TLS fine-tuning. This can be done either via fixed certificates (e.g. when you have your own CA) or with the support of 
Cert Manager.

The specifications of the operator are divided into two sections which control individual workflows.
Note that _any changes on underlying resources that deviate from the specified Plant values will force 
reconciliation until the resources satisfy the requirements._
This is to ensure that both availability and safety measures are respected.

#### Deployment

- `image` (required): specifies the image to use for Deployment containers.
- `containerPort` (optional, defaults to 80): the container port to expose for host traffic.
- `replicas` (optional, defaults to 1): the number of desired pods to deploy.

#### Networking

- `host` (required): the domain name of a network host where the deployed image will be accessible through Ingress.
- `ingressClassName` (optional): the name of the Ingress controller to use.
- `tlsSecretName` (optional): the name of an existing TLS secret to use for Ingress TLS traffic for the given host.
- `tlsCertIssuerRef` (optional): the name of local or cluster _cert-manager_ issuer to use for obtaining 
Ingress TLS certificates for the given host.

Note: You should only specify `tlsSecretName` or `tlsCertIssuerRef` for adding TLS configuration to Ingress, but not both.

### Example
A configurable version depending on the requirements could look something like this:
```yaml
apiVersion: operator.cisco.io/v1
kind: Plant
metadata:
  name: example
spec:
  image: nginx:latest
  host: example.com
  # replicas: 3
  # containerPort: 80
  # ingressClassName: nginx
  # tlsCertIssuerRef:
  #   name: my-issuer
```

## Installation

Before installing Plant Operator, please make sure to install [Cert Manager](https://cert-manager.io/docs/installation/)
dependency which is used for automated certificate handling.

To install Plant Operator, simply run:

```bash
### Install Cert Manager dependency
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml

### Install Plant Operator  
kubectl apply -f https://github.com/fhivemind/plant-operator/releases/download/v1.0.0/plant.yaml
```

You don't need to change any resources of the plant-operator install parameters.
Plant operator resources follows SemVer standard.

## Feature Requests & Contributing
Feel free to open an Issue or a Pull Request if you would like to either request a feature or contribute to the project.

Contributions are always welcome and appreciated! Head out to the discussion forum for more.

## Copyright
Copyright (c) 2023 Ramiz Polic. Available under the Apache 2.0 Licence.