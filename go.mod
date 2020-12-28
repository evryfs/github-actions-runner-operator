module github.com/evryfs/github-actions-runner-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/google/go-github/v32 v32.1.1-0.20200803004443-954e7c82b299
	github.com/gophercloud/gophercloud v0.15.0
	github.com/imdario/mergo v0.3.11
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/redhat-cop/operator-utils v1.0.1
	github.com/stretchr/testify v1.6.1
	github.com/thoas/go-funk v0.7.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-runtime v0.7.0
)
