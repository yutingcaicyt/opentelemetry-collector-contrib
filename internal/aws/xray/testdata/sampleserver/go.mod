module github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/xray/testdata/sampleserver

go 1.19

require github.com/aws/aws-xray-sdk-go v1.8.1

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/aws/aws-sdk-go v1.44.249 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.34.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

retract (
	v0.76.2
	v0.76.1
	v0.65.0
)
