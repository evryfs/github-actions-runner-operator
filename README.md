[![Docker Repository on Quay](https://quay.io/repository/evryfs/github-actions-runner-operator/status "Docker Repository on Quay")](https://quay.io/repository/evryfs/github-actions-runner-operator)
![build](https://github.com/evryfs/github-actions-runner-operator/workflows/build/badge.svg?branch=master)

# github-actions-runner-operator

K8S operator for scheduling github actions runner pods.
[self-hosted-runners](https://help.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
is a way to host your own runners and customize the environment used to run jobs in your GitHub Actions workflows.

This operator helps you schedule runners on-demand in a declarative way.

## Helm-chart based install

Helm3 chart is available from [our existing helm repo](https://github.com/evryfs/helm-charts).

```shell script
helm repo add evryfs-oss https://evryfs.github.io/helm-charts/
kubectl create namespace github-actions-runner-operator
helm install github-actions-runner-operator evryfs-oss/github-actions-runner-operator --namespace github-actions-runner-operator
```

## CRD

Declare a resource like [in the example](deploy/crds/garo.tietoevry.com_v1alpha1_githubactionrunner_cr.yaml)

## Missing parts and weaknesses

* Github's runner-api only exposes the on/off-line status, not if the runner is occupied with a job,
  and hence the scaling does not work properly as intended yet, however I hope this can be implemented, see
  [https://github.com/actions/runner/issues/454]
  
## development

Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) and written in Go.
