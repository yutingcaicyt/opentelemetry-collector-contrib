module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter

go 1.19

require (
	github.com/census-instrumentation/opencensus-proto v0.4.1
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.76.3
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.76.3
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.76.3
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.76.3
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/collector v0.76.1
	go.opentelemetry.io/collector/component v0.76.1
	go.opentelemetry.io/collector/confmap v0.76.1
	go.opentelemetry.io/collector/consumer v0.76.1
	go.opentelemetry.io/collector/exporter v0.76.1
	go.opentelemetry.io/collector/pdata v1.0.0-rcv0011
	go.opentelemetry.io/collector/receiver v0.76.1
	google.golang.org/grpc v1.54.0
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.1.17 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.76.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/cors v1.9.0 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/collector/featuregate v0.76.1 // indirect
	go.opentelemetry.io/collector/semconv v0.76.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.40.0 // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.opentelemetry.io/otel/metric v0.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.14.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => ../../internal/common

replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal => ../../internal/coreinternal

replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent => ../../internal/sharedcomponent

replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus => ../../pkg/translator/opencensus

replace github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver => ../../receiver/opencensusreceiver

retract (
	v0.76.2
	v0.76.1
	v0.65.0
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest => ../../pkg/pdatatest

replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil => ../../pkg/pdatautil
