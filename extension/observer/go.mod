module github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer

go 1.19

require (
	github.com/stretchr/testify v1.8.2
	go.uber.org/zap v1.24.0
)

require (
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.76.2
	v0.76.1
	v0.65.0
)
