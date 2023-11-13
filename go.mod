module github.com/monimesl/bookkeeper-operator

go 1.16

require (
	github.com/go-zookeeper/zk v1.0.2
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/monimesl/operator-helper v0.0.0-20231113132835-3586578317d2
	github.com/onsi/ginkgo/v2 v2.11.0
	github.com/onsi/gomega v1.27.10
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	sigs.k8s.io/controller-runtime v0.16.3
)
