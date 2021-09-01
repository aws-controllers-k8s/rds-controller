module github.com/aws-controllers-k8s/rds-controller

go 1.14

require (
	github.com/aws-controllers-k8s/runtime v0.13.0
	github.com/aws/aws-sdk-go v1.37.10
	github.com/go-logr/logr v0.1.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)
