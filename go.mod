module github.com/redhat-openshift-ecosystem/operator-certification-operator

go 1.16

require (
	github.com/go-git/go-git/v5 v5.4.2
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/operator-framework/api v0.10.5
	github.com/tektoncd/pipeline v0.29.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	sigs.k8s.io/controller-runtime v0.9.2
)
