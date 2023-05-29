# Pure Storage FlashArray Receiver

| Status                   |                     |
| ------------------------ |---------------------|
| Stability                | [in-development]    |
| Supported pipeline types | metrics             |
| Distributions            | [contrib]           |

The Pure Storage FlashArray receiver, receives metrics from Pure Storage internal services hosts.

## Configuration

The following settings are required:
 -  `endpoint` (default: `http://172.0.0.0:9490/metrics/array`): The URL of the scraper selected endpoint

Example:

```yaml
extensions:
  bearertokenauth/array01:
    token: "..."

receivers:
  purefa:
    endpoint: http://172.0.0.1:9490/metrics
    array:
      - address: array01
        auth:
          authenticator: bearertokenauth/array01
    hosts:
      - address: array01
        auth:
          authenticator: bearertokenauth/array01
    directories:
      - address: array01
        auth:
          authenticator: bearertokenauth/array01
    pods:
      - address: array01
        auth:
          authenticator: bearertokenauth/array01
    volumes:
      - address: array01
        auth:
          authenticator: bearertokenauth/array01
    env: dev
    settings:
      reload_intervals:
        array: 10s
        hosts: 13s
        directories: 15s
        pods: 30s
        volumes: 25s
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

[in-development]: https://github.com/open-telemetry/opentelemetry-collector#in-development
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
