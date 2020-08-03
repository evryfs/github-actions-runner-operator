![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/evryfs/github-actions-runner-operator)
![build](https://github.com/evryfs/github-actions-runner-operator/workflows/build/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/evryfs/github-actions-runner-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/evryfs/github-actions-runner-operator)

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

Declare a resource like [in the example](config/samples/garo_v1alpha1_githubactionrunner.yaml)

## Missing parts and weaknesses

* There is a possibility that a runner pod can be deleted while running a build, 
  if it is able to pick a build in the time between listing the api and doing the scaling logic
  
## development

Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) and written in Go.
