# Simple Prometheus Receiver

| Status                   |                |
| ------------------------ | -------------- |
| Stability                | [beta]         |
| Supported pipeline types | metrics        |
| Distributions            | [contrib]      |

The `prometheus_simple` receiver is a wrapper around the [prometheus
receiver](../prometheusreceiver).
This receiver provides a simple configuration interface to configure the
prometheus receiver to scrape metrics from a single target.

## Configuration

The following settings are required:

- `endpoint` (default = `localhost:9090`): The endpoint from which prometheus
metrics should be scraped.

The following settings are optional:

- `collection_interval` (default = `10s`): The internal at which metrics should
be emitted by this receiver.
- `metrics_path` (default = `/metrics`): The path to the metrics endpoint.
- `params` (default = `{}`): The query parameters to pass to the metrics endpoint. If specified, params are appended to `metrics_path` to form the URL with which the target is scraped.
- `use_service_account` (default = `false`): Whether or not to use the
Kubernetes Pod service account for authentication.
- `tls_enabled` (default = `false`): Whether or not to use TLS. Only if
`tls_enabled` is set to `true`, the values under `tls_config` are accounted
for. This setting will be deprecated. Please use `tls` instead.

The `tls_config` section supports the following options. This setting will be deprecated. Please use `tls` instead:

- `ca_file` (no default): Path to the CA cert that has signed the TLS
certificate.
- `cert_file` (no default): Path to the client TLS certificate to use for TLS
required connections.
- `key_file` (no default): Path to the client TLS key to use for TLS required
connections.
- `insecure_skip_verify` (default = `false`): Whether or not to skip
certificate verification.

- `tls`: see [TLS Configuration Settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md#tls-configuration-settings) for the full set of available options.

Example:

```yaml
    receivers:
      prometheus_simple:
        collection_interval: 10s
        use_service_account: true
        endpoint: "172.17.0.5:9153"
        tls:
          ca_file: "/path/to/ca"
          cert_file: "/path/to/cert"
          key_file: "/path/to/key"
          insecure_skip_verify: true
    exporters:
      signalfx:
        access_token: <SIGNALFX_ACCESS_TOKEN>
        url: <SIGNALFX_INGEST_URL>

    service:
      pipelines:
        metrics:
          receivers: [prometheus_simple]
          exporters: [signalfx]
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

[beta]:https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
