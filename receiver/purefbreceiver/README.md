# Pure Storage FlashBlade Receiver

| Status                   |                     |
| ------------------------ |---------------------|
| Stability                | [in-development]    |
| Supported pipeline types | metrics             |
| Distributions            | [contrib]           |

The Pure Storage FlashBlade receiver, receives metrics from Pure Storage FlashBlade via the [Pure Storage FlashBlade OpenMetrics Exporter](https://github.com/PureStorage-OpenConnect/pure-fb-openmetrics-exporter)

## Configuration

The following settings are required:
 -  `endpoint` (default: `http://172.31.60.207:9491/metrics/array`): The URL of the scraper selected endpoint

### Important 

- Only endpoints explicitly added on will be scraped. e.g: `clients`

Example:

```yaml
extensions:
  bearertokenauth/fb01:
    token: "..."

receivers:
  purefb:
    endpoint: http://172.31.60.207:9491/metrics
    arrays:
      - address: fb01
        auth:
          authenticator: bearertokenauth/fb01
    clients:
      - address: fb01
        auth:
          authenticator: bearertokenauth/fb01
    env: dev
    settings:
      reload_intervals:
        array: 5m
        clients: 6m
        usage: 6m

```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

[in-development]: https://github.com/open-telemetry/opentelemetry-collector#in-development
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
