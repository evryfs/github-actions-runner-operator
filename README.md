[![awesome-runners](https://img.shields.io/badge/listed%20on-awesome--runners-blue.svg)](https://github.com/jonico/awesome-runners)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/evryfs/github-actions-runner-operator)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f31ef6cd50994eebb882389ec2ec37f1)](https://app.codacy.com/gh/evryfs/github-actions-runner-operator?utm_source=github.com&utm_medium=referral&utm_content=evryfs/github-actions-runner-operator&utm_campaign=Badge_Grade_Dashboard)
[![Go Report Card](https://goreportcard.com/badge/github.com/evryfs/github-actions-runner-operator)](https://goreportcard.com/report/github.com/evryfs/github-actions-runner-operator)
![build](https://github.com/evryfs/github-actions-runner-operator/workflows/build/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/evryfs/github-actions-runner-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/evryfs/github-actions-runner-operator)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/evryfs/github-actions-runner-operator?sort=semver)
[![Stargazers over time](https://starchart.cc/evryfs/github-actions-runner-operator.svg)](https://starchart.cc/evryfs/github-actions-runner-operator)


# github-actions-runner-operator

K8s operator for scheduling [GitHub Actions](https://github.com/features/actions) runner pods.
[self-hosted-runners](https://help.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
are a way to host your own runners and customize the environment used to run jobs in your GitHub Actions workflows.

This operator helps you scale and schedule runners on-demand in a declarative way.

## Configuration
### Authentication modes

The operator communicates with GitHub in order to determine available jobs and execute workflow on runners. Authentication to GitHub is available using the following modes:

1.  As a [GitHub app](https://docs.github.com/en/free-pro-team@latest/developers/apps/creating-a-github-app).

This is the preferred mode as it provides enhanced security and increased API quota, and avoids exposure of tokens to runner pods.

Follow the guide for creating GitHub applications. There is no need to define a callback url or webhook secret as they are not used by this integration.

Depending on whether the GitHub application will operate at a repository or organization level, the following [permissions](https://docs.github.com/en/free-pro-team@latest/rest/reference/permissions-required-for-github-apps#permission-on-self-hosted-runners) must be set:

* Repository level
    * Actions - Read/Write
    * Administration - Read/Write
* Organization level
    * Self Hosted Runners - Read/Write

Once the GitHub application has been created, obtain the integration ID and download the private key. 

A Github application can only be used by injecting environment variables into the Operator deployment. It is recommended that credentials be stored as Kubernetes secrets and then injected into the operator deployment.

Create a secret called `github-runner-app` by executing the following command in the namespace containing the operator:

```shell script
kubectl create secret generic github-runner-app --from-literal=GITHUB_APP_INTEGRATION_ID=<app_id> --from-file=GITHUB_APP_PRIVATE_KEY=<private_key>
```

Finally define the following on the operator deployment:

```shell script
envFrom:
- secretRef:
    name: github-runner-app
````

2.  Using [Personal Access Tokens (PAT)](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token)

Create a Personal Access token with rights at a repository or organization level.

This PAT can be defined at the operator level or within the custom resource (A PAT defined at the CR level will take precedence)

To make use of a PAT that is declared at a CR level, first create a secret called `actions-runner`

```shell script
kubectl create secret generic actions-runner --from-literal=GH_TOKEN=<token>
```

Define the `tokenRef` field on the `GithubActionRunner` custom resource as shown below:

```yaml
apiVersion: garo.tietoevry.com/v1alpha1
kind: GithubActionRunner
metadata:
  name: runner-pool
spec:
  tokenRef:
    key: GH_TOKEN
    name: actions-runner
```

### Runner Scope

Runners can be registered either against an individual repository or at an organizational level. The following fields are available on the `GithubActionRunner` custom resource to specify the repository and/or organization to monitor actions:

  * `organization` - GitHub user or Organization
  * `repository` - (Optional) GitHub repository

```yaml
apiVersion: garo.tietoevry.com/v1alpha1
kind: GithubActionRunner
metadata:
  name: runner-pool
spec:
  # the github org, required
  organization: yourOrg
  # the githb repository
  repository: myrepo
```

### Runner Selection

Arguably the most important field of the `GithubActionRunner` custom resource is the `podTemplateSpec` field as it allow you to define the runner that will be managed by the operator. You have the flexibility to define all of the properties that will be needed by the runner including the image, resources and environment variables. During normal operation, the operator will create a token that can be used in your runner to communicate with GitHub. This token is created in a secret called `<CR_NAME>-regtoken` in the `RUNNER_TOKEN` key. You should inject this secret into your runner using an environment variable or volume mount.

## Installation Methods

The following options are available to install the operator:
### Helm Chart

A [Helm](https://helm.sh/) chart is available from [this Helm repository](https://github.com/evryfs/helm-charts).

Use the following steps to create a namespace and install the operator into the namespace using a Helm chart

```shell script
helm repo add evryfs-oss https://evryfs.github.io/helm-charts/
kubectl create namespace github-actions-runner-operator
helm install github-actions-runner-operator evryfs-oss/github-actions-runner-operator --namespace github-actions-runner-operator
```
### Manual

Execute the following commands to deploy the operator using manifests available within this repository.

_Note:_ The [Kustomize](https://kustomize.io/) tool is required

1. Install the CRD's

```shell script
make install
```

2. Deploy the Operator

```shell script
make deploy
```

### OperatorHub

Coming Soon

## Examples

A sample of the `GithubActionRunner` custom resource is found [here](config/samples/garo_v1alpha1_githubactionrunner.yaml)

## development

Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) / [Kube builder](https://github.com/kubernetes-sigs/kubebuilder) and written in Go.
