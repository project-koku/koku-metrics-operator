module github.com/project-koku/koku-metrics-operator

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/google/uuid v1.1.2
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v0.0.0-20200117162508-e7ccdda6ba67
	github.com/operator-framework/api v0.2.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.14.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
