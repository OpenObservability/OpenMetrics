# Current Prometheus text format (0.0.4)

Official docs: <https://prometheus.io/docs/instrumenting/exposition_formats/>
Old working document: <https://docs.google.com/document/d/1TapRdI5Tu2e8AIqREtdv_C36t4GGsDQr64KixE-gWu8/edit#>

Note: In-line comments _not_ copied over

Basic format looks like:

    # HELP http_requests_total The total number of HTTP requests.
    # TYPE http_requests_total counter
    http_requests_total{method="post",code="200"} 1027 1395066363000
    http_requests_total{method="post",code="400"}    3 1395066363000

Escaping in label values:

    msdos_file_access_time_seconds{path="C:\\DIR\\FILE.TXT",error="Cannot find file:\n\"FILE.TXT\""} 1.458255915e9

Minimalistic line:

    metric_without_timestamp_and_labels 12.47

A weird metric from before the epoch:

    something_weird{problem="division by zero"} +Inf -3982045

A histogram, which has a pretty complex representation in the text format:

    # HELP http_request_duration_seconds A histogram of the request duration.
    # TYPE http_request_duration_seconds histogram
    http_request_duration_seconds_bucket{le="0.05"} 24054
    http_request_duration_seconds_bucket{le="0.1"} 33444
    http_request_duration_seconds_bucket{le="0.2"} 100392
    http_request_duration_seconds_bucket{le="0.5"} 129389
    http_request_duration_seconds_bucket{le="1"} 133988
    http_request_duration_seconds_bucket{le="+Inf"} 144320
    http_request_duration_seconds_sum 53423
    http_request_duration_seconds_count 144320

Finally a summary, which has a complex representation, too:

    # HELP rpc_duration_seconds A summary of the RPC duration in seconds.
    # TYPE rpc_duration_seconds summary
    rpc_duration_seconds{quantile="0.01"} 3102
    rpc_duration_seconds{quantile="0.05"} 3272
    rpc_duration_seconds{quantile="0.5"} 4773
    rpc_duration_seconds{quantile="0.9"} 9001
    rpc_duration_seconds{quantile="0.99"} 76656
    rpc_duration_seconds_sum 1.7560473e+07
    rpc_duration_seconds_count 2693


# Notes, some of them Prometheus-specific:

- Metric names are `[a-zA-Z_:][a-zA-Z0-9_:]*`
- Label names are `[a-zA-Z0-9_]*`
- Label values are full utf-8, null bytes are permitted
- Metrics should not be exposed with colons, that's for end user use for
  aggregation by convention
- Counter metrics end in _total by convention, counters start at 0 and only go
  up
- Gauges do not have a suffix by convention, can go up and down
- All values are float64
- `\n` is the line terminator
- There's no required ordering on label names, however for performance reasons
  a consistent ordering is encouraged
- Histogram values are cumulative, so the le="0.5" bucket includes the values
  in the previous buckets
- The reasoning behind this is that you can remove buckets and still have
  things work. This is intended to allow for abuse mitigation if someone adds
  too many buckets.
- The values in the le and quantile labels are float64.
- The quantile label can have values from 0 to 1.
- The le label can be any non-NaN value, which means _sum isn't actually a
  counter.
- I, @brian-brazil, am not a fan of this and would prefer to restrict it to
  non-negative values only
- The spec says these must be provided in increasing order
- Summaries are quantiles, not percentiles
- This is a general Prometheus best practice, we deal with things in
  seconds/bytes/ratios and leave converting to something more human readable at
  the display layer. This convention is to avoid a mix of units.
- _sum and _count of histograms/summaries aren't always present, due to some
  instrumentation libraries not supporting them.
- _count of a histogram is always the same as the +Inf bucket.
- TYPE and HELP
  - Are optional
  - Are currently discarded by Prometheus
  - If they are stored, conflicts must be handled (last one wins, etc)
- There's a "untyped" TYPE for when you don't know if something is a counter or
  gauge
- +Inf, -Inf and NaN are supported
- There is a specified MIME type
- By convention, metrics are exposed on /metrics - though this is often not
  followed
- We're starting to add support for selecting specific time series via a url
  parameter ?name[]=timeseries, currently only supported in Python and Java

- There's also a number of other conventions/best practices/patterns about how
  to expose time series to be as useful as possible to query in Prometheus,
  though the principles should largely apply to other systems too. These range
  from simple things like always putting units in the metric name (preferably
  base units like seconds), to things that depend on PromQL features such as
  how to handle non-identity labels you might want to use with a target for
  things like version numbers.

## Places it could be improved in general:

- There's slightly different escaping rules for label names and HELP - we
  should unify this
- Whitespace (spaces or tabs) can appear in any number between tokens - this
  should be tightened up
- Timestamps can be appended to this as millisecond integers - fractional
  seconds would be better to allow for other precisions and be more consistent
  with Prometheus generally.
- Specify exactly which spellings of +Inf/-Inf/Inf/NaN are allowed - probably
  all of them, case insensitive. Different languages produce different
  capitalizations and don't always have the + for +Inf.
- Should we worry about float parsing being otherwise consistent? We haven't
  run into any issues yet.
- Adding an end marker, to help detect if the response got cut off
- Users forgetting the final \n is not uncommon when hand-assembling the
  format, is there anything we can do there?

## Potential expansions

There are things which the format currently doesn't do, that have been talked about. Here's some of them:

- It'd be nice to keep the new format parseable by a Prometheus 0.0.4 parser
- The behaviour of Prometheus when a time series is exposed multiple times with
  different timestamps is explicitly undefined, we'll need to specify this
  (likely sorted oldest first)
- If multiple values at different timestamps are supported, compressing them,
  e.g. into one line, might make sense
- Let's figure out semantics first, micro-optimisations can come later
- Timestamp precision other than milliseconds
- Expose as an integer with fractional seconds/float64?
- What should we do about before 1970?
- 64 bit integer support
- Do we want int64 or uint64? Both?
- Anyone reading this: Are there any use cases for large, high-precision
  negative integers?
- Do we need to indicate to the backend which type a value is (e.g. 0 vs 0.0)?
- If we don’t, what happens when the backend “suddenly” gets float64 when it
  could assume it was [u]int64, before
- Gracefully handling this for systems that only support more limited types
- If we know it's a Counter, can we take it mod 2^53 in systems that only
  support float64?
- 128 bit float/integer support
- Once CPUs support 128 bits natively, it’s natural to expand the format.
  Should this simply be a breaking change (semvar major) or otherwise
  anticipated?
- We're using text, so it'll Just Work I imagine
- Boolean support
- Prometheus convention is false=0, true=1.
- Do we need to add tokens for this, or can we just stick to the convention?
- Special values
- Prometheus 2.0 has a special "stale" value used internally (it's a NaN with a
  particular bit pattern) which we may wish to expose to other Prometheus
  servers. Is this something we should allow (if we end up wanting it), or just
  keep as a Prometheus-specific extension?
- String support
- Debate around whether a first class string value should be added
- One idea is around annotations, that are not part of times series identity
- Would these go with a metric or a time series?
- What are the use cases for these? Are they more log-like?
- The approach at
  https://www.robustperception.io/how-to-have-labels-for-machine-roles/ may be
  an option for annotations
- Gauge with value 1 and a label for each piece of information
- Enum use case, string with limited number of values
- Gauge with one time series per potential value, with 0/1 values?
- Byte string support
- Influx have received a small number of requests for this over the years, not
  inclined to support it currently
- Are there additional metadata fields beyond HELP/TYPE that we should
  have?
