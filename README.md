![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/evryfs/github-actions-runner-operator)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f31ef6cd50994eebb882389ec2ec37f1)](https://app.codacy.com/gh/evryfs/github-actions-runner-operator?utm_source=github.com&utm_medium=referral&utm_content=evryfs/github-actions-runner-operator&utm_campaign=Badge_Grade_Dashboard)
[![Go Report Card](https://goreportcard.com/badge/github.com/evryfs/github-actions-runner-operator)](https://goreportcard.com/report/github.com/evryfs/github-actions-runner-operator)
![build](https://github.com/evryfs/github-actions-runner-operator/workflows/build/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/evryfs/github-actions-runner-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/evryfs/github-actions-runner-operator)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/evryfs/github-actions-runner-operator?sort=semver)
[![Stargazers over time](https://starchart.cc/evryfs/github-actions-runner-operator.svg)](https://starchart.cc/evryfs/github-actions-runner-operator)


# github-actions-runner-operator

K8s operator for scheduling github actions runner pods.
[self-hosted-runners](https://help.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
is a way to host your own runners and customize the environment used to run jobs in your GitHub Actions workflows.

This operator helps you scale and schedule runners on-demand in a declarative way.

## Helm-chart based install

Helm3 chart is available from [our existing helm repo](https://github.com/evryfs/helm-charts).

```shell script
helm repo add evryfs-oss https://evryfs.github.io/helm-charts/
kubectl create namespace github-actions-runner-operator
helm install github-actions-runner-operator evryfs-oss/github-actions-runner-operator --namespace github-actions-runner-operator
```

## CRD

Declare a resource like [in the example](config/samples/garo_v1alpha1_githubactionrunner.yaml)

## Weaknesses

  * There is a theoretical possibility that a runner pod can be deleted while running a build,
    if it is able to pick a build in the time between listing the api and doing the scaling logic.

## development

Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) / [Kube builder](https://github.com/kubernetes-sigs/kubebuilder) and written in Go.
