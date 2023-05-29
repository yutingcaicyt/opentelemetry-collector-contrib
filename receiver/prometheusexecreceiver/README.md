# Deprecated prometheus_exec Receiver

| Status                   |              |
| ------------------------ | ------------ |
| Stability                | [deprecated] |
| Supported pipeline types | metrics      |
| Distributions            | none         |

This receiver has been deprecated due to security concerns around the ability to specify the execution of
any arbitrary processes via its configuration. See [#6722](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/6722) for additional details.

This receiver makes it easy for a user to collect metrics from third-party
services **via Prometheus exporters**. It's meant for people who want a
plug-and-play solution to getting metrics from those third-party services
that sometimes simply don't natively export metrics or speak any
instrumentation protocols (MySQL, Apache, Nginx, JVM, etc.) while taking
advantage of the large [Prometheus
exporters](https://prometheus.io/docs/instrumenting/exporters/) ecosystem.

Through the configuration file, you can indicate which binaries to run
(usually [Prometheus
exporters](https://prometheus.io/docs/instrumenting/exporters/), which are
custom binaries that expose the third-party services' metrics using the
Prometheus protocol) and `prometheus_exec` will take care of starting the
specified binaries with their equivalent Prometheus receiver. This receiver
also supports starting binaries with flags and environment variables,
retrying them with exponential backoff if they crash, string templating, and
random port assignments.

> :information_source: If you do not need to spawn the binaries locally,
please consider using the [core Prometheus
receiver](../prometheusreceiver)
or the [Simple Prometheus
receiver](../simpleprometheusreceiver).

## Configuration

For each `prometheus_exec` defined in the configuration file, the specified
command will be run. The command *should* start a binary that exposes
Prometheus metrics and an equivalent Prometheus receiver will be instantiated
to scrape its metrics, if configured correctly.

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

The following settings are required:

- `exec` (no default): The string of the command to be run, with any flags
needed. The format should be: `directory/binary_to_run flag1 flag2`.

The following settings are optional:

- `env` (no default): To use environment variables, under the `env` key
should be a list of key (`name`) - value (`value`) pairs. They are
case-sensitive. When running a command, these environment variables are added
to the pre-existing environment variables the Collector is currently running
with.
- `scrape_interval` (default = `60s`): How long the delay between scrapes
done by the receiver is.
- `port` (no default): A number indicating the port the receiver should be
scraping the binary's metrics from.

Two important notes about `port`:

1. If it is omitted, we will try to randomly generate a port
for you, and retry until we find one that is free. Beware when using this,
since you also need to indicate your binary to listen on that same port with
the use of a flag and string templating inside the command, which is covered
in 2.

2. **All** instances of `{{port}}` in any string of any key for the enclosing
`prometheus_exec` will be replaced with either the port value indicated or
the randomly generated one if no port value is set with the `port` key.
String templating of `{{port}}` is supported in `exec`, `custom_name` and
`env`.

Example:

```yaml
receivers:
    # this receiver will listen on port 9117
    prometheus_exec/apache:
        exec: ./apache_exporter
        port: 9117

    # this receiver will listen on port 9187 and {{port}} inside the command will become 9187
    prometheus_exec/postgresql:
        exec: ./postgres_exporter --web.listen-address=:{{port}}
        port: 9187

    # this receiver will listen on a random port and that port will be substituting the {{port}} inside the command
    prometheus_exec/mysql:
        exec: ./mysqld_exporter --web.listen-address=:{{port}}
        scrape_interval: 60s
        env:
          - name: DATA_SOURCE_NAME
            value: user:password@(hostname:port)/dbname
          - name: SECONDARY_PORT
            value: {{port}}
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

[deprecated]:https://github.com/open-telemetry/opentelemetry-collector#deprecated
