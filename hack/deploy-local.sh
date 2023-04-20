#!/usr/bin/env bash

# recreate cluster if needed
if [ -n "$RECREATE" ]; then
  kind delete cluster
  ./hack/kind-with-registry.sh
  dupk
fi

# recreate stack
make all
make docker-build docker-push IMG=localhost:5001/plant:latest
make deps
kubectl apply -f config/samples/example_deps.yaml

# deploy
make undeploy
make deploy IMG=localhost:5001/plant:latest
