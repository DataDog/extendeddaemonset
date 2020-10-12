module github.com/DataDog/extendeddaemonset

go 1.13

require (
	github.com/blang/semver v3.5.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/google/go-cmp v0.4.0
	github.com/hako/durafmt v0.0.0-20200710122514-c0fb7b4da026
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/common v0.6.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.10.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/cli-runtime v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kube-state-metrics v1.9.7
	sigs.k8s.io/controller-runtime v0.6.2
)
