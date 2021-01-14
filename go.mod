module github.com/evryfs/github-actions-runner-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/google/go-github/v33 v33.0.0
	github.com/gophercloud/gophercloud v0.15.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/imdario/mergo v0.3.11
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/palantir/go-githubapp v0.6.0
	github.com/redhat-cop/operator-utils v1.1.1
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.7.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-runtime v0.7.0
)
