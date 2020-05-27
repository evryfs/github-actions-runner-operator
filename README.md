[![Docker Repository on Quay](https://quay.io/repository/evryfs/github-actions-runner-operator/status "Docker Repository on Quay")](https://quay.io/repository/evryfs/github-actions-runner-operator)
![build](https://github.com/evryfs/github-actions-runner-operator/workflows/build/badge.svg?branch=master)

# github-actions-runner-operator

K8S operator for scheduling github actions runner pods.
[self-hosted-runners](https://help.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
is a way to host your own runners and customize the environment used to run jobs in your GitHub Actions workflows.

This operator helps you schedule runners on-demand in a declarative way.

## CRD

Declare a resource like this:
```yaml
apiVersion: "garo.tietoevry.com/v1alpha1"
kind: GithubActionRunner
metadata:
  name: runner
spec:
  # minimum amount of runners that should be available
  minRunners: 1
  # maximum amount of runners that should be available
  maxRunners: 1
  # the GitHub organization name, from https://github.com/yourGithubOrgId
  organization: yourGithubOrgId
  # reference to secret in same namespace containing the token for GitHub, needs org-level scope
  tokenRef:
    # name of secret
    name: github-token
    # key within secret holding the value
    key: GH_TOKEN
  # a podspec like you wish for the runners.
  # the spec here will run the one from https://github.com/evryfs/github-actions-runner as the runner,
  # with a companion dind (Docker In Docker) container. 
  podSpec:
    containers:
      - env:
          - name: RUNNER_DEBUG
            value: "true"
          - name: DOCKER_TLS_CERTDIR
            value: /certs
          - name: DOCKER_HOST
            value: tcp://localhost:2376
          - name: DOCKER_TLS_VERIFY
            value: "1"
          - name: DOCKER_CERT_PATH
            value: /certs/client
          - name: GH_ORG
            value: yourGithubOrgId
        envFrom:
          - secretRef:
              name: github-token
        image: quay.io/evryfs/github-actions-runner:latest
        imagePullPolicy: Always
        lifecycle:
          preStop:
            exec:
              command:
                - /bin/bash
                - -c
                - /remove_runner.sh
        name: runner
        resources: {}
        volumeMounts:
          - mountPath: /certs
            name: docker-certs
          - mountPath: /settings-xml
            name: settings-xml
      - env:
          - name: DOCKER_TLS_CERTDIR
            value: /certs
        image: docker:stable-dind
        imagePullPolicy: Always
        name: docker
        resources: {}
        securityContext:
          privileged: true
        volumeMounts:
          - mountPath: /var/lib/docker
            name: docker-storage
          - mountPath: /certs
            name: docker-certs
    volumes:
      - emptyDir: {}
        name: docker-storage
      - emptyDir: {}
        name: docker-certs
      - configMap:
          defaultMode: 420
          name: settings-xml
        name: settings-xml
```

## Helm-chart

Helm3 chart is available from [our existing helm repo](https://github.com/evryfs/helm-charts).

```shell script
helm repo add evryfs-oss https://evryfs.github.io/helm-charts/
kubectl create namespace github-actions-runner-operator
helm install github-actions-runner-operator evryfs-oss/github-actions-runner-operator --namespace github-actions-runner-operator
```

## Missing parts and weaknesses

* Github's runner-api only exposes the on/off-line status, not if the runner is occupied with a job,
  and hence the scaling does not work properly as intended yet, however I hope this can be implemented, see
  [https://github.com/actions/runner/issues/454]
  
## development

Operator is based on [Operator SDK](https://github.com/operator-framework/operator-sdk) and written in Go.
