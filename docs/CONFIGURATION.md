<!-- markdownlint-disable MD024 -->

# Configuration

Telegraf's configuration file is written using [TOML][] and is composed of
three sections: [global tags][], [agent][] settings, and [plugins][].

## Generating a Configuration File

A default config file can be generated by telegraf:

```sh
telegraf config > telegraf.conf
```

To generate a file with specific inputs and outputs, you can use the
--input-filter and --output-filter flags:

```sh
telegraf config --input-filter cpu:mem:net:swap --output-filter influxdb:kafka
```

[View the full list][flags] of Telegraf commands and flags or by running
`telegraf --help`.

### Windows PowerShell v5 Encoding

In PowerShell 5, the default encoding is UTF-16LE and not UTF-8. Telegraf
expects a valid UTF-8 file. This is not an issue with PowerShell 6 or newer,
as well as the Command Prompt or with using the Git Bash shell.

As such, users will need to specify the output encoding when generating a full
configuration file:

```sh
telegraf.exe config | Out-File -Encoding utf8 telegraf.conf
```

This will generate a UTF-8 encoded file with a BOM. However, Telegraf can
handle the leading BOM.

## Configuration Loading

The location of the configuration file can be set via the `--config` command
line flag.

When the `--config-directory` command line flag is used files ending with
`.conf` in the specified directory will also be included in the Telegraf
configuration.

On most systems, the default locations are `/etc/telegraf/telegraf.conf` for
the main configuration file and `/etc/telegraf/telegraf.d` for the directory of
configuration files.

## Environment Variables

Environment variables can be used anywhere in the config file, simply surround
them with `${}`.  Replacement occurs before file parsing. For strings
the variable must be within quotes, e.g., `"${STR_VAR}"`, for numbers and booleans
they should be unquoted, e.g., `${INT_VAR}`, `${BOOL_VAR}`.

Users need to keep in mind that when using double quotes the user needs to
escape any backslashes (e.g. `"C:\\Program Files"`) or other special characters.
If using an environment variable with a single backslash, then enclose the
variable in single quotes which signifies a string literal (e.g.
`'C:\Program Files'`).

In addition to this, Telegraf also supports Shell parameter expansion for
environment variables which allows syntax such as:

- `${VARIABLE:-default}` evaluates to default if VARIABLE is unset or empty in
                         the environment.
- `${VARIABLE-default}` evaluates to default only if VARIABLE is unset in the
                        environment. Similarly, the following syntax allows you
                        to specify mandatory variables:
- `${VARIABLE:?err}` exits with an error message containing err if VARIABLE is
                     unset or empty in the environment.
- `${VARIABLE?err}` exits with an error message containing err if VARIABLE is
                     unset in the environment.

When using the `.deb` or `.rpm` packages, you can define environment variables
in the `/etc/default/telegraf` file.

**Example**:

`/etc/default/telegraf`:

For InfluxDB 1.x:

```shell
USER="alice"
INFLUX_URL="http://localhost:8086"
INFLUX_SKIP_DATABASE_CREATION="true"
INFLUX_PASSWORD="monkey123"
```

For InfluxDB OSS 2:

```shell
INFLUX_HOST="http://localhost:8086" # used to be 9999
INFLUX_TOKEN="replace_with_your_token"
INFLUX_ORG="your_username"
INFLUX_BUCKET="replace_with_your_bucket_name"
```

For InfluxDB Cloud 2:

```shell
# For AWS West (Oregon)
INFLUX_HOST="https://us-west-2-1.aws.cloud2.influxdata.com"
# Other Cloud URLs at https://v2.docs.influxdata.com/v2.0/reference/urls/#influxdb-cloud-urls
INFLUX_TOKEN=”replace_with_your_token”
INFLUX_ORG="yourname@yourcompany.com"
INFLUX_BUCKET="replace_with_your_bucket_name"
```

`/etc/telegraf.conf`:

```toml
[global_tags]
  user = "${USER}"

[[inputs.mem]]

# For InfluxDB 1.x:
[[outputs.influxdb]]
  urls = ["${INFLUX_URL}"]
  skip_database_creation = ${INFLUX_SKIP_DATABASE_CREATION}
  password = "${INFLUX_PASSWORD}"

# For InfluxDB OSS 2:
[[outputs.influxdb_v2]]
  urls = ["${INFLUX_HOST}"]
  token = "${INFLUX_TOKEN}"
  organization = "${INFLUX_ORG}"
  bucket = "${INFLUX_BUCKET}"

# For InfluxDB Cloud 2:
[[outputs.influxdb_v2]]
  urls = ["${INFLUX_HOST}"]
  token = "${INFLUX_TOKEN}"
  organization = "${INFLUX_ORG}"
  bucket = "${INFLUX_BUCKET}"
```

The above files will produce the following effective configuration file to be
parsed:

```toml
[global_tags]
  user = "alice"

[[inputs.mem]]

# For InfluxDB 1.x:
[[outputs.influxdb]]
  urls = "http://localhost:8086"
  skip_database_creation = true
  password = "monkey123"

# For InfluxDB OSS 2:
[[outputs.influxdb_v2]]
  urls = ["http://127.0.0.1:8086"] # double check the port. could be 9999 if using OSS Beta
  token = "replace_with_your_token"
  organization = "your_username"
  bucket = "replace_with_your_bucket_name"

# For InfluxDB Cloud 2:
[[outputs.influxdb_v2]]
  # For AWS West (Oregon)
  INFLUX_HOST="https://us-west-2-1.aws.cloud2.influxdata.com"
  # Other Cloud URLs at https://v2.docs.influxdata.com/v2.0/reference/urls/#influxdb-cloud-urls
  token = "replace_with_your_token"
  organization = "yourname@yourcompany.com"
  bucket = "replace_with_your_bucket_name"
```

## Secret-store secrets

Additional or instead of environment variables, you can use secret-stores
to fill in credentials or similar. To do so, you need to configure one or more
secret-store plugin(s) and then reference the secret in your plugin
configurations. A reference to a secret is specified in form
`@{<secret store id>:<secret name>}`, where the `secret store id` is the unique
ID you defined for your secret-store and `secret name` is the name of the secret
to use.
**NOTE:** Both, the `secret store id` as well as the `secret name` can only
consist of letters (both upper- and lowercase), numbers and underscores.

**Example**:

This example illustrates the use of secret-store(s) in plugins

```toml
[global_tags]
  user = "alice"

[[secretstores.os]]
  id = "local_secrets"

[[secretstores.jose]]
  id = "cloud_secrets"
  path = "/etc/telegraf/secrets"
  # Optional reference to another secret store to unlock this one.
  password = "@{local_secrets:cloud_store_passwd}"

[[inputs.http]]
  urls = ["http://server.company.org/metrics"]
  username = "@{local_secrets:company_server_http_metric_user}"
  password = "@{local_secrets:company_server_http_metric_pass}"

[[outputs.influxdb_v2]]
  urls = ["https://us-west-2-1.aws.cloud2.influxdata.com"]
  token = "@{cloud_secrets:influxdb_token}"
  organization = "yourname@yourcompany.com"
  bucket = "replace_with_your_bucket_name"
```

### Notes

When using plugins supporting secrets, Telegraf locks the memory pages
containing the secrets. Therefore, the locked memory limit has to be set to a
suitable value. Telegraf will check the limit and the number of used secrets at
startup and will warn if your limit is too low. In this case, please increase
the limit via `ulimit -l`.

If you are running Telegraf in an jail you might need to allow locked pages in
that jail by setting `allow.mlock = 1;` in your config.

## Intervals

Intervals are durations of time and can be specified for supporting settings by
combining an integer value and time unit as a string value.  Valid time units are
`ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

```toml
[agent]
  interval = "10s"
```

## Global Tags

Global tags can be specified in the `[global_tags]` table in key="value"
format. All metrics that are gathered will be tagged with the tags specified.
Global tags are overridden by tags set by plugins.

```toml
[global_tags]
  dc = "us-east-1"
```

## Agent

The agent table configures Telegraf and the defaults used across all plugins.

- **interval**: Default data collection [interval][] for all inputs.

- **round_interval**: Rounds collection interval to [interval][]
  ie, if interval="10s" then always collect on :00, :10, :20, etc.

- **metric_batch_size**:
  Telegraf will send metrics to outputs in batches of at most
  metric_batch_size metrics.
  This controls the size of writes that Telegraf sends to output plugins.

- **metric_buffer_limit**:
  Maximum number of unwritten metrics per output.  Increasing this value
  allows for longer periods of output downtime without dropping metrics at the
  cost of higher maximum memory usage. Oldest metrics are overwritten in favor
  of new ones when the buffer fills up.

- **collection_jitter**:
  Collection jitter is used to jitter the collection by a random [interval][].
  Each plugin will sleep for a random time within jitter before collecting.
  This can be used to avoid many plugins querying things like sysfs at the
  same time, which can have a measurable effect on the system.

- **collection_offset**:
  Collection offset is used to shift the collection by the given [interval][].
  This can be be used to avoid many plugins querying constraint devices
  at the same time by manually scheduling them in time.

- **flush_interval**:
  Default flushing [interval][] for all outputs. Maximum flush_interval will be
  flush_interval + flush_jitter.

- **flush_jitter**:
  Default flush jitter for all outputs. This jitters the flush [interval][]
  by a random amount. This is primarily to avoid large write spikes for users
  running a large number of telegraf instances. ie, a jitter of 5s and interval
  10s means flushes will happen every 10-15s.

- **precision**:
  Collected metrics are rounded to the precision specified as an [interval][].

  Precision will NOT be used for service inputs. It is up to each individual
  service input to set the timestamp at the appropriate precision.

- **debug**:
  Log at debug level.

- **quiet**:
  Log only error level messages.

- **logformat**:
  Log format controls the way messages are logged and can be one of "text",
  "structured" or, on Windows, "eventlog". The output file (if any) is
  determined by the `logfile` setting.

- **structured_log_message_key**:
  Message key for structured logs, to override the default of "msg".
  Ignored if `logformat` is not "structured".

- **logfile**:
  Name of the file to be logged to or stderr if unset or empty. This
  setting is ignored for the "eventlog" format.

- **logfile_rotation_interval**:
  The logfile will be rotated after the time interval specified.  When set to
  0 no time based rotation is performed.

- **logfile_rotation_max_size**:
  The logfile will be rotated when it becomes larger than the specified size.
  When set to 0 no size based rotation is performed.

- **logfile_rotation_max_archives**:
  Maximum number of rotated archives to keep, any older logs are deleted.  If
  set to -1, no archives are removed.

- **log_with_timezone**:
  Pick a timezone to use when logging or type 'local' for local time. Example: 'America/Chicago'.
  [See this page for options/formats.](https://socketloop.com/tutorials/golang-display-list-of-timezones-with-gmt)

- **hostname**:
  Override default hostname, if empty use os.Hostname()

- **omit_hostname**:
  If set to true, do no set the "host" tag in the telegraf agent.

- **snmp_translator**:
  Method of translating SNMP objects. Can be "netsnmp" (deprecated) which
  translates by calling external programs `snmptranslate` and `snmptable`,
  or "gosmi" which translates using the built-in gosmi library.

- **statefile**:
  Name of the file to load the states of plugins from and store the states to.
  If uncommented and not empty, this file will be used to save the state of
  stateful plugins on termination of Telegraf. If the file exists on start,
  the state in the file will be restored for the plugins.

- **always_include_local_tags**:
  Ensure tags explicitly defined in a plugin will *always* pass tag-filtering
  via `taginclude` or `tagexclude`. This removes the need to specify local tags
  twice.

- **always_include_global_tags**:
  Ensure tags explicitly defined in the `global_tags` section will *always* pass
  tag-filtering   via `taginclude` or `tagexclude`. This removes the need to
  specify those tags twice.

- **skip_processors_after_aggregators**:
  By default, processors are run a second time after aggregators. Changing
  this setting to true will skip the second run of processors.

- **buffer_strategy**:
  The type of buffer to use for telegraf output plugins. Supported modes are
  `memory`, the default and original buffer type, and `disk`, an experimental
  disk-backed buffer which will serialize all metrics to disk as needed to
  improve data durability and reduce the chance for data loss. This is only
  supported at the agent level.

- **buffer_directory**:
  The directory to use when in `disk` buffer mode. Each output plugin will make
  another subdirectory in this directory with the output plugin's ID.

## Plugins

Telegraf plugins are divided into 4 types: [inputs][], [outputs][],
[processors][], and [aggregators][].

Unlike the `global_tags` and `agent` tables, any plugin can be defined
multiple times and each instance will run independently.  This allows you to
have plugins defined with differing configurations as needed within a single
Telegraf process.

Each plugin has a unique set of configuration options, reference the
sample configuration for details.  Additionally, several options are available
on any plugin depending on its type.

### Input Plugins

Input plugins gather and create metrics.  They support both polling and event
driven operation.

Parameters that can be used with any input plugin:

- **alias**: Name an instance of a plugin.
- **interval**:
  Overrides the `interval` setting of the [agent][Agent] for the plugin.  How
  often to gather this metric. Normal plugins use a single global interval, but
  if one particular input should be run less or more often, you can configure
  that here.
- **precision**:
  Overrides the `precision` setting of the [agent][Agent] for the plugin.
  Collected metrics are rounded to the precision specified as an [interval][].

  When this value is set on a service input, multiple events occurring at the
  same timestamp may be merged by the output database.
- **time_source**:
  Specifies the source of the timestamp on metrics. Possible values are:
  - `metric` will not alter the metric (default)
  - `collection_start` sets the timestamp to when collection started
  - `collection_end` set the timestamp to when collection finished

  `time_source` will NOT be used for service inputs. It is up to each individual
  service input to set the timestamp.
- **collection_jitter**:
  Overrides the `collection_jitter` setting of the [agent][Agent] for the
  plugin.  Collection jitter is used to jitter the collection by a random
  [interval][]. The value must be non-zero to override the agent setting.
- **collection_offset**:
  Overrides the `collection_offset` setting of the [agent][Agent] for the
  plugin. Collection offset is used to shift the collection by the given
  [interval][]. The value must be non-zero to override the agent setting.
- **name_override**: Override the base name of the measurement.  (Default is
  the name of the input).
- **name_prefix**: Specifies a prefix to attach to the measurement name.
- **name_suffix**: Specifies a suffix to attach to the measurement name.
- **tags**: A map of tags to apply to a specific input's measurements.
- **log_level**: Override the log-level for this plugin. Possible values are
  `error`, `warn`, `info`, `debug` and `trace`.

The [metric filtering][] parameters can be used to limit what metrics are
emitted from the input plugin.

#### Examples

Use the name_suffix parameter to emit measurements with the name `cpu_total`:

```toml
[[inputs.cpu]]
  name_suffix = "_total"
  percpu = false
  totalcpu = true
```

Use the name_override parameter to emit measurements with the name `foobar`:

```toml
[[inputs.cpu]]
  name_override = "foobar"
  percpu = false
  totalcpu = true
```

Emit measurements with two additional tags: `tag1=foo` and `tag2=bar`

> **NOTE**: With TOML, order matters.  Parameters belong to the last defined
> table header, place `[inputs.cpu.tags]` table at the *end* of the plugin
> definition.

```toml
[[inputs.cpu]]
  percpu = false
  totalcpu = true
  [inputs.cpu.tags]
    tag1 = "foo"
    tag2 = "bar"
```

Alternatively, when using the inline table syntax, the tags do not need
to go at the end:

```toml
[[inputs.cpu]]
  tags = {tag1 = "foo", tag2 = "bar"}
  percpu = false
  totalcpu = true
```

Utilize `name_override`, `name_prefix`, or `name_suffix` config options to
avoid measurement collisions when defining multiple plugins:

```toml
[[inputs.cpu]]
  percpu = false
  totalcpu = true

[[inputs.cpu]]
  percpu = true
  totalcpu = false
  name_override = "percpu_usage"
  fieldexclude = ["cpu_time*"]
```

### Output Plugins

Output plugins write metrics to a location.  Outputs commonly write to
databases, network services, and messaging systems.

Parameters that can be used with any output plugin:

- **alias**: Name an instance of a plugin.
- **flush_interval**: The maximum time between flushes.  Use this setting to
  override the agent `flush_interval` on a per plugin basis.
- **flush_jitter**: The amount of time to jitter the flush interval.  Use this
  setting to override the agent `flush_jitter` on a per plugin basis. The value
  must be non-zero to override the agent setting.
- **metric_batch_size**: The maximum number of metrics to send at once.  Use
  this setting to override the agent `metric_batch_size` on a per plugin basis.
- **metric_buffer_limit**: The maximum number of unsent metrics to buffer.
  Use this setting to override the agent `metric_buffer_limit` on a per plugin
  basis.
- **name_override**: Override the original name of the measurement.
- **name_prefix**: Specifies a prefix to attach to the measurement name.
- **name_suffix**: Specifies a suffix to attach to the measurement name.
- **log_level**: Override the log-level for this plugin. Possible values are
  `error`, `warn`, `info` and `debug`.

The [metric filtering][] parameters can be used to limit what metrics are
emitted from the output plugin.

#### Examples

Override flush parameters for a single output:

```toml
[agent]
  flush_interval = "10s"
  flush_jitter = "5s"
  metric_batch_size = 1000

[[outputs.influxdb]]
  urls = [ "http://example.org:8086" ]
  database = "telegraf"

[[outputs.file]]
  files = [ "stdout" ]
  flush_interval = "1s"
  flush_jitter = "1s"
  metric_batch_size = 10
```

### Processor Plugins

Processor plugins perform processing tasks on metrics and are commonly used to
rename or apply transformations to metrics.  Processors are applied after the
input plugins and before any aggregator plugins.

Parameters that can be used with any processor plugin:

- **alias**: Name an instance of a plugin.
- **order**: The order in which the processor(s) are executed. starting with 1.
  If this is not specified then processor execution order will be the order in
  the config. Processors without "order" will take precedence over those
  with a defined order.
- **log_level**: Override the log-level for this plugin. Possible values are
  `error`, `warn`, `info` and `debug`.

The [metric filtering][] parameters can be used to limit what metrics are
handled by the processor.  Excluded metrics are passed downstream to the next
processor.

#### Examples

If the order processors are applied matters you must set order on all involved
processors:

```toml
[[processors.rename]]
  order = 1
  [[processors.rename.replace]]
    tag = "path"
    dest = "resource"

[[processors.strings]]
  order = 2
  [[processors.strings.trim_prefix]]
    tag = "resource"
    prefix = "/api/"
```

### Aggregator Plugins

Aggregator plugins produce new metrics after examining metrics over a time
period, as the name suggests they are commonly used to produce new aggregates
such as mean/max/min metrics.  Aggregators operate on metrics after any
processors have been applied.

Parameters that can be used with any aggregator plugin:

- **alias**: Name an instance of a plugin.
- **period**: The period on which to flush & clear each aggregator. All
  metrics that are sent with timestamps outside of this period will be ignored
  by the aggregator.
  The default period is set to 30 seconds.
- **delay**: The delay before each aggregator is flushed. This is to control
  how long for aggregators to wait before receiving metrics from input
  plugins, in the case that aggregators are flushing and inputs are gathering
  on the same interval.
  The default delay is set to 100 ms.
- **grace**: The duration when the metrics will still be aggregated
  by the plugin, even though they're outside of the aggregation period. This
  is needed in a situation when the agent is expected to receive late metrics
  and it's acceptable to roll them up into next aggregation period.
  The default grace duration is set to 0 s.
- **drop_original**: If true, the original metric will be dropped by the
  aggregator and will not get sent to the output plugins.
- **name_override**: Override the base name of the measurement.  (Default is
  the name of the input).
- **name_prefix**: Specifies a prefix to attach to the measurement name.
- **name_suffix**: Specifies a suffix to attach to the measurement name.
- **tags**: A map of tags to apply to the measurement - behavior varies based on aggregator.
- **log_level**: Override the log-level for this plugin. Possible values are
  `error`, `warn`, `info` and `debug`.

The [metric filtering][] parameters can be used to limit what metrics are
handled by the aggregator.  Excluded metrics are passed downstream to the next
aggregator.

#### Examples

Collect and emit the min/max of the system load1 metric every 30s, dropping
the originals.

```toml
[[inputs.system]]
  fieldinclude = ["load1"] # collects system load1 metric.

[[aggregators.minmax]]
  period = "30s"        # send & clear the aggregate every 30s.
  drop_original = true  # drop the original metrics.

[[outputs.file]]
  files = ["stdout"]
```

Collect and emit the min/max of the swap metrics every 30s, dropping the
originals. The aggregator will not be applied to the system load metrics due
to the `namepass` parameter.

```toml
[[inputs.swap]]

[[inputs.system]]
  fieldinclude = ["load1"] # collects system load1 metric.

[[aggregators.minmax]]
  period = "30s"        # send & clear the aggregate every 30s.
  drop_original = true  # drop the original metrics.
  namepass = ["swap"]   # only "pass" swap metrics through the aggregator.

[[outputs.file]]
  files = ["stdout"]
```

## Metric Filtering

Metric filtering can be configured per plugin on any input, output, processor,
and aggregator plugin.  Filters fall under two categories: Selectors and
Modifiers.

### Selectors

Selector filters include or exclude entire metrics.  When a metric is excluded
from a Input or an Output plugin, the metric is dropped.  If a metric is
excluded from a Processor or Aggregator plugin, it is skips the plugin and is
sent onwards to the next stage of processing.

- **namepass**:
An array of [glob pattern][] strings. Only metrics whose measurement name
matches a pattern in this list are emitted. Additionally, custom list of
separators can be specified using `namepass_separator`. These separators
are excluded from wildcard glob pattern matching.

- **namedrop**:
The inverse of `namepass`. If a match is found the metric is discarded. This
is tested on metrics after they have passed the `namepass` test. Additionally,
custom list of separators can be specified using `namedrop_separator`. These
separators are excluded from wildcard glob pattern matching.

- **tagpass**:
A table mapping tag keys to arrays of [glob pattern][] strings.  Only metrics
that contain a tag key in the table and a tag value matching one of its
patterns is emitted. This can either use the explicit table syntax (e.g.
a subsection using a `[...]` header) or inline table syntax (e.g like
a JSON table with `{...}`). Please see the below notes on specifying the table.

- **tagdrop**:
The inverse of `tagpass`.  If a match is found the metric is discarded. This
is tested on metrics after they have passed the `tagpass` test.

> NOTE: Due to the way TOML is parsed, when using the explicit table
> syntax (with `[...]`) for `tagpass` and `tagdrop` parameters, they
> must be defined at the **end** of the plugin definition, otherwise subsequent
> plugin config options will be interpreted as part of the tagpass/tagdrop
> tables.
> NOTE: When using the inline table syntax (e.g. `{...}`) the table must exist
> in the main plugin definition and not in any sub-table (e.g.
> `[[inputs.win_perf_counters.object]]`).

- **metricpass**:
A ["Common Expression Language"][CEL] (CEL) expression with boolean result where
`true` will allow the metric to pass, otherwise the metric is discarded. This
filter expression is more general compared to e.g. `namepass` and also allows
for time-based filtering. An introduction to the CEL language can be found
[here][CEL intro]. Further details, such as available functions and expressions,
are provided in the [language definition][CEL lang] as well as in the
[extension documentation][CEL ext].

**NOTE:** Expressions that may be valid and compile, but fail at runtime will
result in the expression reporting as `true`. The metrics will pass through
as a result. An example is when reading a non-existing field. If this happens,
the evaluation is aborted, an error is logged, and the expression is reported as
`true`, so the metric passes.

> NOTE: As CEL is an *interpreted* languguage, this type of filtering is much
> slower compared to `namepass`/`namedrop` and friends. So consider to use the
> more restricted filter options where possible in case of high-throughput
> scenarios.

[CEL]:https://github.com/google/cel-go/tree/master
[CEL intro]: https://codelabs.developers.google.com/codelabs/cel-go
[CEL lang]: https://github.com/google/cel-spec/blob/master/doc/langdef.md
[CEL ext]: https://github.com/google/cel-go/tree/master/ext#readme

### Modifiers

Modifier filters remove tags and fields from a metric. If all fields are
removed the metric is removed and as result not passed through to the following
processors or any output plugin. Tags and fields are modified before a metric is
passed to a processor, aggregator, or output plugin. When used with an input
plugin the filter applies after the input runs.

- **fieldinclude**:
An array of [glob pattern][] strings.  Only fields whose field key matches a
pattern in this list are emitted.

- **fieldexclude**:
The inverse of `fieldinclude`.  Fields with a field key matching one of the
patterns will be discarded from the metric.  This is tested on metrics after
they have passed the `fieldinclude` test.

- **taginclude**:
An array of [glob pattern][] strings.  Only tags with a tag key matching one of
the patterns are emitted.  In contrast to `tagpass`, which will pass an entire
metric based on its tag, `taginclude` removes all non matching tags from the
metric.  Any tag can be filtered including global tags and the agent `host`
tag.

- **tagexclude**:
The inverse of `taginclude`. Tags with a tag key matching one of the patterns
will be discarded from the metric.  Any tag can be filtered including global
tags and the agent `host` tag.

### Filtering Examples

#### Using tagpass and tagdrop

```toml
[[inputs.cpu]]
  percpu = true
  totalcpu = false
  fieldexclude = ["cpu_time"]
  # Don't collect CPU data for cpu6 & cpu7
  [inputs.cpu.tagdrop]
    cpu = [ "cpu6", "cpu7" ]

[[inputs.disk]]
  [inputs.disk.tagpass]
    # tagpass conditions are OR, not AND.
    # If the (filesystem is ext4 or xfs) OR (the path is /opt or /home)
    # then the metric passes
    fstype = [ "ext4", "xfs" ]
    # Globs can also be used on the tag values
    path = [ "/opt", "/home*" ]

[[inputs.win_perf_counters]]
  [[inputs.win_perf_counters.object]]
    ObjectName = "Network Interface"
    Instances = ["*"]
    Counters = [
      "Bytes Received/sec",
      "Bytes Sent/sec"
    ]
    Measurement = "win_net"
  # Do not send metrics where the Windows interface name (instance) begins with
  # 'isatap' or 'Local'
  [inputs.win_perf_counters.tagdrop]
    instance = ["isatap*", "Local*"]
```

#### Using fieldinclude and fieldexclude

```toml
# Drop all metrics for guest & steal CPU usage
[[inputs.cpu]]
  percpu = false
  totalcpu = true
  fieldexclude = ["usage_guest", "usage_steal"]

# Only store inode related metrics for disks
[[inputs.disk]]
  fieldinclude = ["inodes*"]
```

#### Using namepass and namedrop

```toml
# Drop all metrics about containers for kubelet
[[inputs.prometheus]]
  urls = ["http://kube-node-1:4194/metrics"]
  namedrop = ["container_*"]

# Only store rest client related metrics for kubelet
[[inputs.prometheus]]
  urls = ["http://kube-node-1:4194/metrics"]
  namepass = ["rest_client_*"]
```

#### Using namepass and namedrop with separators

```toml
# Pass all metrics of type 'A.C.B' and drop all others like 'A.C.D.B'
[[inputs.socket_listener]]
  data_format = "graphite"
  templates = ["measurement*"]

  namepass = ["A.*.B"]
  namepass_separator = "."

# Drop all metrics of type 'A.C.B' and pass all others like 'A.C.D.B'
[[inputs.socket_listener]]
  data_format = "graphite"
  templates = ["measurement*"]

  namedrop = ["A.*.B"]
  namedrop_separator = "."
```

#### Using taginclude and tagexclude

```toml
# Only include the "cpu" tag in the measurements for the cpu plugin.
[[inputs.cpu]]
  percpu = true
  totalcpu = true
  taginclude = ["cpu"]

# Exclude the "fstype" tag from the measurements for the disk plugin.
[[inputs.disk]]
  tagexclude = ["fstype"]
```

#### Metrics can be routed to different outputs using the metric name and tags

```toml
[[outputs.influxdb]]
  urls = [ "http://localhost:8086" ]
  database = "telegraf"
  # Drop all measurements that start with "aerospike"
  namedrop = ["aerospike*"]

[[outputs.influxdb]]
  urls = [ "http://localhost:8086" ]
  database = "telegraf-aerospike-data"
  # Only accept aerospike data:
  namepass = ["aerospike*"]

[[outputs.influxdb]]
  urls = [ "http://localhost:8086" ]
  database = "telegraf-cpu0-data"
  # Only store measurements where the tag "cpu" matches the value "cpu0"
  [outputs.influxdb.tagpass]
    cpu = ["cpu0"]
```

#### Routing metrics to different outputs based on the input

Metrics are tagged with `influxdb_database` in the input, which is then used to
select the output.  The tag is removed in the outputs before writing with `tagexclude`.

```toml
[[outputs.influxdb]]
  urls = ["http://influxdb.example.com"]
  database = "db_default"
  [outputs.influxdb.tagdrop]
    influxdb_database = ["*"]

[[outputs.influxdb]]
  urls = ["http://influxdb.example.com"]
  database = "db_other"
  tagexclude = ["influxdb_database"]
  [outputs.influxdb.tagpass]
    influxdb_database = ["other"]

[[inputs.disk]]
  [inputs.disk.tags]
    influxdb_database = "other"
```

## Transport Layer Security (TLS)

Reference the detailed [TLS][] documentation.

[TOML]: https://github.com/toml-lang/toml#toml
[global tags]: #global-tags
[interval]: #intervals
[agent]: #agent
[plugins]: #plugins
[inputs]: #input-plugins
[outputs]: #output-plugins
[processors]: #processor-plugins
[aggregators]: #aggregator-plugins
[metric filtering]: #metric-filtering
[TLS]: /docs/TLS.md
[glob pattern]: https://github.com/gobwas/glob#syntax
[flags]: /docs/COMMANDS_AND_FLAGS.md
