module github.com/GoogleCloudPlatform/opentelemetry-operations-collector

go 1.14

require (
	github.com/StackExchange/wmi v0.0.0-20181212234831-e0a55b97c705 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/hashicorp/consul/api v1.4.0 // indirect
	github.com/hashicorp/serf v0.9.2 // indirect
	github.com/mitchellh/go-testing-interface v1.0.3 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/stackdriverexporter v0.5.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.6.1-0.20200723171718-a2ff1aa6779e
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor v0.6.0
	github.com/opentracing/opentracing-go v1.1.1-0.20190913142402-a7454ce5950e // indirect
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v2.20.6+incompatible // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae
	k8s.io/client-go v8.0.0+incompatible // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
