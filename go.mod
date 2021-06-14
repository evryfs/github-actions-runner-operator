module github.com/evryfs/github-actions-runner-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/google/go-github/v35 v35.3.0
	github.com/gophercloud/gophercloud v0.18.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/palantir/go-githubapp v0.7.0
	github.com/redhat-cop/operator-utils v1.1.2
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.8.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.0
)
