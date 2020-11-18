# pinta-scheduler Helm chart

This Helm chart contains necessary configurations to start up Pinta on a Kubernetes cluster.

Install using Helm chart:

```
$ kubectl create namespace pinta-system
namespace/pinta-system created
```
```
$ helm repo add pinta [URL_TO_PINTA_REPO]
$ helm install pinta pinta/pinta-scheduler -n pinta-system
```
