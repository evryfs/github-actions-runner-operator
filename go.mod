module github.com/evryfs/github-actions-runner-operator

go 1.15

require (
	github.com/go-logr/logr v0.2.0
	github.com/google/go-github/v32 v32.1.1-0.20200803004443-954e7c82b299
	github.com/gophercloud/gophercloud v0.13.0
	github.com/imdario/mergo v0.3.11
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/stretchr/testify v1.6.1
	github.com/thoas/go-funk v0.7.0
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/controller-runtime v0.6.3
)
