# Snowflake Receiver

| Status                   |               |
|--------------------------|---------------|
| Stability                | [development] |
| Supported pipeline types | metrics       |
| Distributions            | [contrib]     |

This receiver collects metrics from a Snowflake account by connecting to and querying a Snowflake deployment.

## Configuration

The following settings are required:

* `username` (no default): Specifies username used to authenticate with Snowflake.
* `password` (no default): Specifies the password associated with designated username. Used to authenticate with Snowflake.
* `account` (no default): Specifies the account from which metrics are to be gathered.
* `warehouse` (no default): Specifies the warehouse, or unit of computer, designated for the metric gathering queries. Must be an existing warehouse in your Snowflake account.

The following settings are optional:

* `metrics` (default: see `DefaultMetricSettings` [here](./internal/metadata/generated_metrics.go)): Controls the enabling/disabling of specific metrics. For in depth documentation on the allowable metrics see [here](./documentation.md).
* `schema` (default: 'ACCOUNT_USAGE'): Snowflake DB schema containing usage statistics and metadata to be monitored.
* `database` (default: 'SNOWFLAKE'): Snowflake DB containing schema with usage statistics and metadata to be monitored.
* `role` (default: 'ACCOUNTADMIN'): Role associated with the username designated above. By default admin privileges are required to access most/all of the usage data.
* `collection_interval` (default: 30m): Collection interval for metrics receiver. The value for this setting must be readable by golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration).

Example:
```yaml
receivers:
  snowflake:
    username: snowflakeuser
    password: securepassword
    account: bigbusinessaccount
    warehouse: metricWarehouse
    collection_interval: 18m
    metrics:
      snowflake.database.bytes_scanned.avg:
        enabled: true
      snowflake.database.bytes_deketed.avg:
        enabled: false
```

The full list of settings exposed for this receiver are documented [here](./config.go) with a detailed sample configuration [here](./testdata/config.yaml)

[development]: https://github.com/open-telemetry/opentelemetry-collector#development
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
