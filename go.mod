module github.com/chenleji/pipeline-operator

go 1.12

require (
	github.com/astaxie/beego v1.11.1
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/knative/build v0.6.0
	github.com/knative/pkg v0.0.0-20190624141606-d82505e6c5b4
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.4.2
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/pkg/errors v0.8.1
	github.com/robfig/cron v1.2.0
	golang.org/x/net v0.0.0-20181114220301-adae6a3d119a
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
)
