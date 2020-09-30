#!/bin/bash
if ! command -v kubectl &> /dev/null
then
    curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /bin/
fi

if ! command -v kind &> /dev/null
then
    curl -sLo kind "$(curl -sL https://api.github.com/repos/kubernetes-sigs/kind/releases/latest | jq -r '[.assets[] | select(.name == "kind-linux-amd64")] | first | .browser_download_url')"
    chmod +x kind
    sudo mv kind /bin/
    kind create cluster
fi

export KUBECONFIG=$HOME/.kind/config

kind create cluster --config ./hack/kind_cluster.yaml  --kubeconfig $KUBECONFIG

kubectl apply -f ./artifacts/examples/crd.yaml
kubectl wait --timeout=5m --for=condition=Established crd $(kubectl get crd --output=jsonpath='{.items[*].metadata.name}')


echo -e '====================\nTo use the cluster:\nexport KUBECONFIG='$KUBECONFIG '\n===================='
