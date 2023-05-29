# Cassandra Exporter

| Status                   |              |
|--------------------------|--------------|
| Stability                | [alpha]      |
| Supported pipeline types | logs, traces |
| Distributions            | [contrib]    |

## Configuration options

The following settings can be optionally configured:

- `dsn` The Cassandra server DSN (Data Source Name), for example `127.0.0.1`.
  reference: [https://pkg.go.dev/github.com/gocql/gocql](https://pkg.go.dev/github.com/gocql/gocql)
- `keyspace` (default = otel): The keyspace name.
- `trace_table` (default = otel_spans): The table name for traces.
- `replication` (default = class: SimpleStrategy, replication_factor: 1) The strategy of
  replication. https://cassandra.apache.org/doc/4.1/cassandra/architecture/dynamo.html#replication-strategy
- `compression` (default = LZ4Compressor) https://cassandra.apache.org/doc/latest/cassandra/operating/compression.html

## Example

```yaml
exporters:
  cassandra:
    dsn: 127.0.0.1
    keyspace: "otel"
    trace_table: "otel_spans"
    replication:
      class: "SimpleStrategy"
      replication_factor: 1
    compression:
      algorithm: "ZstdCompressor"
```