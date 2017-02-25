# Pirate

## What is Pirate?

Pirate is a gateway, written in go, which accepts client-side metrics via UDP and makes them available to [grafsy](https://github.com/leoleovich/grafsy).
In the end you can have near-time dashboards with client-side metrics.


## Architecture

![Pirate Server Architecture](doc/architecture.png)

- UDP packets are accepted by the UDP server
- if compression is enabled, the packets are decompressed for the next step
- plain text messages are parsed into header attributes and the metrics body (see [protocol](#protocol))
- parsed messages are validated, invalid messages or metrics will be dropped (see [validation](#validation))
- valid metric names are then resolved to their Graphite path
- at the end the writer will take care of sending the metrics to Grafsy via TCP or filesystem
- beside this there is a monitoring component, which tracks own metrics like `udp_received`, `metrics_received` or `metrics_dropped`,
  which are also sent to the writer

All the components of the data pipeline are implemented as workers (goroutines), which scale with the amount of CPUs. Additionally
there are buffers (go channels) between those components. This means, even if some messages are processed a bit slower,
the rest of the system should not be affected.

In the rare case that all buffers should be full, incoming UDP packets will be dropped immediately.


## Protocol

The client's message is a (GZIP-encoded) UDP packet, which consist of two parts: the header and the body.

The header is the first line of the message and contains information about the whole message (e.g. the project identifier and custom attributes).
The body contains the metrics, where each metric consists of a name, a numeric value and a timestamp.

### Example

In plain text (before GZIP-compression is applied) the message could look like this:
```
project=awesome_game; version=1.3.37;
fps 55 1234567890
fps 48 1234567900
memory_usage 209715200 1234567890
fps 53 1234567950
memory_usage 205215400 1234568320
```

Here are two attributes defined by the header. The `project` is always required, because it determines, which rules
are applied to the metrics. The `version` attribute is a custom attribute, which can be used in the `graphite_path`
configuration via `{attr.version}`. Additional attributes, which are not used in the `graphite_path` will be ignored.

In total there are 5 metrics, which will be processed according to the configuration of the `awesome_game` project.


## Output Format

According to the project rules the metric name will be resolved to a graphite path (see [placeholders](#placeholders)
for more information). The resulting rows could then look like this:

```
games.awesome_game.client.ios.1_3_37.fps 55 1234567890
games.awesome_game.client.ios.1_3_37.fps 48 1234567900
games.awesome_game.client.ios.1_3_37.memory_usage 209715200 1234567890
games.awesome_game.client.ios.1_3_37.fps 53 1234567950
games.awesome_game.client.ios.1_3_37.memory_usage 205215400 1234568320
```


## Validation

The incoming packages and the single metrics are validated. This includes:

- the project identifier from the header must be configured, otherwise the message is dropped
- all values of custom header fields must match their configured regex, otherwise the message is dropped
- sent metric names must be configured, otherwise the metric is dropped
- metric values must be within the configured min/max range to be valid, otherwise the metric is dropped
- timestamp validation: metrics with future timestamps or too old timestamps (> 1h) get dropped

All metrics which passed this validation will be processed and sent to Grafsy


## Configuration

### General

| Key                  | Description                                              |
|----------------------|----------------------------------------------------------|
| `udp_address`        | The address to listen for UDP packages                   |
| `graphite_target`    | The target, where the graphite data should be sent to, e.g. `tcp://localhost:3002` or `file:///tmp/metrics.log` |
| `monitoring_enabled` | Whether Pirate should generate own monitoring metrics (received metrics, dropped metrics, etc.) |
| `monitoring_path`    | Graphite path to use for monitoring metrics, only used when monitoring enabled (may contain [placeholders](#placeholders)) |
| `gzip`               | Whether to use GZIP compressed messages |
| `log_level`          | The log level (debug, info, notice, warning, error, critical) |

### Projects

Every project has its own custom sub-section within the configuration file under the key `projects.PROJECT_ID`,
where `PROJECT_ID` is your own identifier, which is used from the message header to determine the target project

The projects sub-section then has the following keys:

| Key               | Description                                              |
|-------------------|----------------------------------------------------------|
| `graphite_path`   | The path each incoming metric is written to. It might contain placeholders (see [placeholders](#placeholders) for more information) |
| `attributes`      | Custom attributes, which can be used within [placeholders](#placeholders) |
| `metrics`         | Allowed metric definitions with boundary check           |

### Attributes

Every project may have custom attribute definitions under the key `projects.PROJECT_ID.attributes.ATTRIBUTE_ID`,
where `ATTRIBUTE_ID` is your own identifier, which is used from the message header (analogous to the `project` attribute).

The value of every attribute is just a regular expression, which is tested on incoming message headers. If at least one
of the attribute's expressions does not match, the whole message will be dropped.

### Metrics

Every project may have one or more metric definitions under the key `projects.PROJECT_ID.metrics.METRIC_ID`,
where `METRIC_ID` is your own metric identifier, which equals the metric name from the message body.

This definition includes a min and max value, which are used for boundary validation.
Metrics, which are out of the configured boundary, will be dropped.
Optionally the Graphite Path of the project can be overridden for single metrics.

The metrics sub-section supports the following options:

| Key             | Description                                              |
|-----------------|----------------------------------------------------------|
| `graphite_path` | The Graphite path, which is used for this metric. This is optional: if left out, the `graphite_path` from the project is used |
| `min`           | The minimum allowed value (float32) |
| `max`           | The maximum allowed value (float32) |

### Placeholders

Within your `graphite_path` configuration you can use two types of placeholders: attributes (`attr`) and metrics (`metric`).
The first one relates to attributes, which are sent with the message header and contain the project ID and arbitrary data.
The `metric` variable relates to the metric itself and currently only allows access to the metric sub-key `name`

Example:
```yaml
projects:
  example_project:
    graphite_path: games.awesome_game.client.{attr.platform}.{attr.version}.{metric.name}
```

Sending the following message
```
project=example_project; platform=ios; version=1.3.37;
fps 55 1234567890
```

would result in the path `games.awesome_game.client.ios.1_3_37.fps`

*Notice:* during the placeholder resolution all dots are substituted by underscores in order to not influence your graphite path hierarchy

If one of the attributes is missing, the metrics won't be processed any further

### Full Config Example
```yaml
udp_address: 0.0.0.0:33333
graphite_target: tcp://127.0.0.1:3002
monitoring_enabled: true
monitoring_path: games.awesome_game.pirate.{metric.name}
gzip: true
log_level: debug # debug mode is very verbose and should only be used for - well - debugging purpose :)
projects:
  awesome_client:
    graphite_path: AVG.games.awesome_game.client.{attr.platform}.{attr.version}.{metric.name}
    attributes:
      platform: ^(ios|android)$
      version: ^[0-9]+\.[0-9]+$
    metrics:
      frames_per_second:
        min: 0
        max: 60
      memory_usage:
        min: 0
        max: 2147483648
      startup_time:
        min: 0
        max: 90
      errors:
        graphite_path: SUM.games.awesome_game.client.{attr.platform}.{attr.version}.{metric.name}
        min: 0
        max: 1000
  awesome_backend:
    graphite_path: servers.some_project.{attr.hostname}.{metric.name}
    attributes:
      hostname: ^[0-9a-f]{12}$
    metrics:
      requests_per_second:
        min: 0
        max: 100000
      response_time:
        min: 0
        max: 100
```