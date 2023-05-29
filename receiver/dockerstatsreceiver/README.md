# Docker Stats Receiver

| Status                   |           |
| ------------------------ |-----------|
| Stability                | [alpha]   |
| Supported pipeline types | metrics   |
| Distributions            | [contrib] |

The Docker Stats receiver queries the local Docker daemon's container stats API for
all desired running containers on a configured interval.  These stats are for container
resource usage of cpu, memory, network, and the
[blkio controller](https://www.kernel.org/doc/Documentation/cgroup-v1/blkio-controller.txt).

> :information_source: Requires Docker API version 1.22+ and only Linux is supported.

## Configuration

The following settings are required:

- `endpoint` (default = `unix:///var/run/docker.sock`): Address to reach the desired Docker daemon.

The following settings are optional:

- `collection_interval` (default = `10s`): The interval at which to gather container stats.
- `container_labels_to_metric_labels` (no default): A map of Docker container label names whose label values to use
as the specified metric label key.
- `env_vars_to_metric_labels` (no default): A map of Docker container environment variables whose values to use
as the specified metric label key.
- `excluded_images` (no default, all running containers monitored): A list of strings,
[regexes](https://golang.org/pkg/regexp/), or [globs](https://github.com/gobwas/glob) whose referent container image
names will not be among the queried containers. `!`-prefixed negations are possible for all item types to signify that
only unmatched container image names should be excluded.
    - Regexes must be placed between `/` characters: `/my?egex/`.  Negations are to be outside the forward slashes:
    `!/my?egex/` will exclude all containers whose name doesn't match the compiled regex `my?egex`.
    - Globs are non-regex items (e.g. `/items/`) containing any of the following: `*[]{}?`.  Negations are supported:
    `!my*container` will exclude all containers whose image name doesn't match the blob `my*container`.
- `timeout` (default = `5s`): The request timeout for any docker daemon query.
- `api_version` (default = `1.22`): The Docker client API version (must be 1.22+). [Docker API versions](https://docs.docker.com/engine/api/).
- `metrics` (defaults at [./documentation.md](./documentation.md)): Enables/disables individual metrics. See [./documentation.md](./documentation.md) for full detail.

Example:

```yaml
receivers:
  docker_stats:
    endpoint: http://example.com/
    collection_interval: 2s
    timeout: 20s
    api_version: 1.24
    container_labels_to_metric_labels:
      my.container.label: my-metric-label
      my.other.container.label: my-other-metric-label
    env_vars_to_metric_labels:
      MY_ENVIRONMENT_VARIABLE: my-metric-label
      MY_OTHER_ENVIRONMENT_VARIABLE: my-other-metric-label
    excluded_images:
      - undesired-container
      - /.*undesired.*/
      - another-*-container
    metrics: 
      container.cpu.usage.percpu:
        enabled: true
      container.network.io.usage.tx_dropped:
        enabled: false
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

[alpha]: https://github.com/open-telemetry/opentelemetry-collector#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib

### Migrating from ScraperV1 to ScraperV2

*Note: These changes are now in effect and ScraperV1 have been removed as of v0.71.*

There are some breaking changes from ScraperV1 to ScraperV2. The work done for these changes is tracked in [#9794](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/9794).

| Breaking Change                     | Action                                                                  |
|-------------------------------------|-------------------------------------------------------------------------|
| Many metrics are no longer emitted by default. | See [documentation.md](./documentation.md) to see which metrics are enabled by default. Enable/disable as desired. |
| BlockIO metrics names changed. The type of operation is no longer in the metric name suffix, and is now in an attribute. For example `container.blockio.io_merged_recursive.read` becomes `container.blockio.io_merged_recursive` with an `operation:read` attribute. | Be aware of the metric name changes and make any adjustments to what your downstream expects from BlockIO metrics. |
| Memory metrics measured in Bytes are now [non-monotonic sums](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/metrics/data-model.md#opentelemetry-protocol-data-model-consumer-recommendations) instead of gauges. | Most likely there is no action. The aggregation type is different but the values are the same. Be aware of how your downstream handles gauges vs non-monotonic sums. |
| Config option `provide_per_core_cpu_metrics` has been removed. | Enable the `container.cpu.usage.percpu` metric as per [documentation.md](./documentation.md). |
