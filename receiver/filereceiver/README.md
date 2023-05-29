# File Receiver

| Status                   |                       |
|--------------------------|-----------------------|
| Stability                | [development]         |
| Supported pipeline types | metrics, traces, logs |
| Distributions            | [contrib]             |

The File Receiver reads the output of a
[File Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/fileexporter),
converting that output to metrics, and sending the metrics down the pipeline.

Currently, the only file format supported is the File Exporter's JSON format. Reading compressed output, rotated files,
or telemetry other than metrics are not supported at this time.

## Getting Started

The following setting is required:

- `path` [no default]: the file in the same format as written by a File Exporter.

The following setting is optional:

- `throttle` [default: 1]: a determines how fast telemetry is replayed. A value of `0` means
  that it will be replayed as fast as the system will allow. A value of `1` means that it will
  be replayed at the same rate as the data came in, as indicated by the timestamps on the
  input file's telemetry data. Higher values mean that the replay speed will be slower by a
  multiple of the throttle value. Values can be decimals, e.g. `0.5` means that telemetry will be
  replayed at 2x the rate indicated by the telemetry's timestamps.

## Example

```yaml
receivers:
  file:
    path: my-telemetry-file
    throttle: 0.5
```
