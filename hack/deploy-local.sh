#!/usr/bin/env bash

# recreate cluster if needed
if [ -n "$RECREATE" ]; then
  kind delete cluster
  ./hack/kind-with-registry.sh

  make deps
  kubectl apply -f config/samples/example_deps.yaml
fi

# recreate stack
make all
make docker-build docker-push IMG=localhost:5001/plant:latest

# deploy
make undeploy
make deploy IMG=localhost:5001/plant:latest
