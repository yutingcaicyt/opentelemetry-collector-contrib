# Count Connector

| Status                   |                                                           |
|------------------------- |---------------------------------------------------------- |
| Stability                | [in development]                                          |
| Supported pipeline types | See [Supported Pipeline Types](#supported-pipeline-types) |
| Distributions            | []                                                        |

The `count` connector can be used to count spans, span events, metrics, data points, and log records.

## Supported Pipeline Types

| [Exporter Pipeline Type] | [Receiver Pipeline Type] |
| ------------------------ | ------------------------ |
| traces                   | metrics                  |
| metrics                  | metrics                  |
| logs                     | metrics                  |

## Configuration

If you are not already familiar with connectors, you may find it helpful to first visit the [Connectors README].

### Default Configuration

The `count` connector may be used without any configuration settings. The following table describes the
default behavior of the connector.

| [Exporter Pipeline Type] | Description                         | Default Metric Names                         |
| ------------------------ | ------------------------------------| -------------------------------------------- |
| traces                   | Counts all spans and span events.   | `trace.span.count`, `trace.span.event.count` |
| metrics                  | Counts all metrics and data points. | `metric.count`, `metric.data_point.count`    |
| logs                     | Counts all log records.             | `log.record.count`                           |

For example, in the following configuration the connector will count spans and span events from the `traces/in`
pipeline and emit metrics called `trace.span.count` and `trace.span.event.count` onto the `metrics/out` pipeline.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:

service:
  pipelines:
    traces/in:
      receivers: [foo]
      exporters: [count]
    metrics/out:
      receivers: [count]
      exporters: [bar]
```

### Custom Counts

Optionally, emit custom counts by defining metrics under one or more of the following sections:

- `spans`
- `spanevents`
- `metrics`
- `datapoints`
- `logs`

Optionally, specify a description for the metric.

Note: If any custom metrics are defined for a data type, the default metric will not be emitted.

#### Conditions

Conditions may be specified for custom metrics. If specified, data that matches any one
of the conditions will be counted. i.e. Conditions are ORed together.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
    spanevents:
      my.prod.event.count:
        description: The number of span events from my prod environment.
        conditions:
          - 'attributes["env"] == "prod"'
          - 'name == "prodevent"'
```

#### Attributes

`spans`, `spanevents`, `datapoints`, and `logs` may be counted according to attributes.

If attributes are specified for custom metrics, a separate count will be generated for each unique
set of attribute values. Each count will be emitted as a data point on the same metric.

Optionally, include a `default_value` for an attribute, to count data that does not contain the attribute.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
    logs:
      my.log.count:
        description: The number of logs from each environment.
        attributes:
          - key: env
            default_value: unspecified_environment
```

### Example Usage

Count spans and span events, only exporting the count metrics.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [count]
    metrics:
      receivers: [count]
      exporters: [bar]
```

Count spans and span events, exporting both the original traces and the count metrics.

```yaml
receivers:
  foo:
exporters:
  bar/traces_backend:
  bar/metrics_backend:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [bar/traces_backend, count]
    metrics:
      receivers: [count]
      exporters: [bar/metrics_backend]
```

Count spans, span events, metrics, data points, and log records, exporting count metrics to a separate backend.

```yaml
receivers:
  foo/traces:
  foo/metrics:
  foo/logs:
exporters:
  bar/all_types:
  bar/counts_only:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo/traces]
      exporters: [bar/all_types, count]
    metrics:
      receivers: [foo/metrics]
      exporters: [bar/all_types, count]
    logs:
      receivers: [foo/logs]
      exporters: [bar/all_types, count]
    metrics/counts:
      receivers: [count]
      exporters: [bar/counts_only]
```

Count logs with a severity of ERROR or higher.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
    logs:
      my.error.log.count:
        description: Error+ logs.
        conditions:
          - `severity_number >= SEVERITY_NUMBER_ERROR`
service:
  pipelines:
    logs:
      receivers: [foo]
      exporters: [count]
    metrics:
      receivers: [count]
      exporters: [bar]
```

Count logs with a severity of ERROR or higher. Maintain a separate count for each environment.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
    logs:
      my.error.log.count:
        description: Error+ logs.
        conditions:
          - `severity_number >= SEVERITY_NUMBER_ERROR`
        attributes:
          - key: env
service:
  pipelines:
    logs:
      receivers: [foo]
      exporters: [count]
    metrics:
      receivers: [count]
      exporters: [bar]
```

Count all spans and span events (default behavior). Count metrics and data points based on the `env` attribute.

```yaml
receivers:
  foo/traces:
  foo/metrics:
  foo/logs:
exporters:
  bar/all_types:
  bar/counts_only:
connectors:
  count:
    metrics:
      my.prod.metric.count:
        conditions:
         - `attributes["env"] == "prod"
      my.test.metric.count:
        conditions:
         - `attributes["env"] == "test"
    datapoints:
      my.prod.datapoint.count:
        conditions:
         - `attributes["env"] == "prod"
      my.test.datapoint.count:
        conditions:
         - `attributes["env"] == "test"
service:
  pipelines:
    traces:
      receivers: [foo/traces]
      exporters: [bar/all_types, count]
    metrics:
      receivers: [foo/metrics]
      exporters: [bar/all_types, count]
    metrics/counts:
      receivers: [count]
      exporters: [bar/counts_only]
```

[in development]:https://github.com/open-telemetry/opentelemetry-collector#in-development
[Connectors README]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md
[Exporter Pipeline Type]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#exporter-pipeline-type
[Receiver Pipeline Type]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#receiver-pipeline-type
[OTTL Syntax]:https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/README.md