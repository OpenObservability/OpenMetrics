# OpenMetrics Face-to-Face

2017-10-10 - 2017-10-11

Google London

# Attendees
* Richard “RichiH” Hartmann richih@richih.org - RichiH@GitHub
* Pim van Pelt pim@google.com - pimvanpelt@GitHub
* Ben Kochie superq@gmail.com - SuperQ@GitHub
* Wilmer van der Gaast wilmer@google.com  - Wilm0r@GitHub
* Tom Wilkie tom.wilkie@gmail.com - tomwilkie@GitHub
* Brian Brazil brian.brazil@gmail.com - brian-brazil@GitHub
* Paul Dix paul@influxdb.com - pauldix@GitHub
* Eric Silva ericsilva@google.com ericanthonysilva@GitHub
* Jeromy Carriere jcarriere@google.com - sjcarriere2@GitHub
* Matt Bostock matt@mattbostock.com - mattbostock@GitHub
* Morgan McLean morganmclean@google.com - mtwo@GitHub
* Sumeer Bhola sbhola@google.com - sumeer@GitHub
* Chris Larsen clarsen@oath.com - manolama@GitHub
* Bogdan Drutu bdrutu@google.com - bogdandrutu@Github
* Dino Oliva dpo@google.com - dinooliva@GitHub

# Content
## Tuesday 2017-10-10 11:00

* Introductions
  * Pim: Tasked with integrating SRE teams and standardizing our approaches on production between SRE, GCP and Developer Infrastructure, including Borgmon and Monarch; primary responsibilities: Releases/Rollouts (CI/CD), Systems Modeling and Actuation (CA), and Monitoring/Alerting for google’s production stacks. I use Prometheus headed by Grafana for www.sixxs.net, a ~60 server deployment, and other private IoT based projects.
  * Ben: Ex-Googler, joined SoundCloud and works on Prometheus. Wants Open Source Streamz - surprised that the industry still hasn’t grokked /varz.
  * Wilmer: Long-time Borgmon having spent years hearing how Borgmon sucks from people who never tried outside offerings, grateful that Prometheus now exists.
  * Tom: ex-Google, ex-Weaveworks, now Prometheus team and Kausal founder (Kausal runs hosted Prometheus)
  * Richi: never worked for Google… way too many years of being on call, want to push proper metrics into the world  with the help of an RFC
  * Brian: ex-Googler, SRE and and one point helped defined the corp based borgmons. Became borgmon-guy, ended up knowing its power and deficiencies. Now core developer on Prometheus, avoiding new warts and removing others. Person for PromQL and client library semantics.
  * Paul Dix: Founded InfluxDB, want to support Prometheus metrics as a first class citizen
  * Eric: Product manager of Stackdriver
  * Jeromy: Engineering director for Monarch and its external manifestation in stackdriver. Want to make Google’s cloud offering as open as possible, exposing our infrastructure in APIs and a la carte (so customers can pick and choose what they’d like to use, and replace what they don’t like / want).
  * Matt Bostock, Cloudflare. Now 190 instances of Prometheus. Also use OpenTSDB.
  * Morgan Mclane - OpenCensus PM,
  * Chris Larsen - Monitoring engineer @ Yahoo/Oath, member of the valley large scale monitoring group meeting with companies like Paypal, Facebook, Ebay, Apple, LinkedIn, etc. Maintainer and lead on OpenTSDB
  * Bogdan: OpenCensus library team lead
* Brian: Prometheus Data Model
  * It’s Borgmon, not MySQL
    * Not for things which live less than five minutes, Prometheus is not an eventlogger.
    * Jeromy: We need quicker than that for things like Cloud Functions and AppEngine
    * Brian: Need event logs or statsd
    * Jeromy: we do push metrics from appengine; its works…
    * Sumeer: We push at the start, every minute, and at the end
    * RichiH: No matter how, how long, or how often: the exposition format can be the same
  * Label sets are the identity, new label set, new time series
  * Implicit namespacing with underscores
  * Uses float64 for values and int64 millis for timestamps
  * Unix timestamps, base is seconds
  * Counters only go up, can do +x (but can be fractional increments); gauges go up/down
  * Have non-enforced suffixes for metric names
  * Terminology is inconsistent internally within Prometheus, suggested:
    * ‘a_count’: time series name
    * ‘a’: metric
    * ‘a_count{foo=”bar”}’: time series
    * ‘a{foo=”bar”}’: child
    * _sum/_count suffixes used by convention in Prometheus; should they be metadata?
* (evacuation alarm in LON building) -- resumed at 1pm
  * Units
    * Worked in one group that did milliseconds for a decade; then a new project got onboarded, which did microseconds and broke dashboards etc.. This is horrible, so we do seconds.
    * Float64 as a baseline? Other types could be used if desired
    * String annotations: why not make these metrics ending in _info a primary type (metadata) with value 1?
    * Enums: what are they used for?
  * Nagios alert states, DB connection state etc
* Lots of back and forth: Rough consensus: We talked about Prometheus-specific stuff and where this comes from; this does not mandate doing it this way within OpenMetrics
  * Slides https://docs.google.com/presentation/d/1TF1Y7XqgZThb9MUUvJUdEEbVjkNhQ1Ddg54ysTFPgqU/edit?usp=sharing
* Chris: Yamas
  * Seconds, no ms, for timestamp.  Values are float64
  * Common dimensions for set of named metrics
  * Status codes and status msg
  * AOL argus
  * Timestamp is ISO 8601
  * First class units
  * Binary blobs for digests/sketches
  * Annotations stored in line with timeseries
  * Would like to see support for blobs in OpenMetrics
  * Annotation labels which don’t change the identity of a timeseries
  * Slides https://docs.google.com/presentation/d/1UN88du2-mKyEi3F2n_ZFTpQm_yJr1KOQmnoOLPbEhJU/edit?usp=sharing
* Sumeer: Monarch
  * Successor to Borgmon
  * Multi-tenant
  * Path-based, people should claim subtrees to maintain
  * 2 instances; google and public cloud
  * Supported units are from Unified Code for Units of Measure
  * Trace IDs can be attached to histogram buckets
  * Buckets can change and will be interpolated at change boundaries if changed
  * Timestamps are in microseconds
  * Open issues:
    * Memory use in clients when storing distributions
    * Specifying bucket sizes when metrics are produced
* Census
  * Google’s instrumentation code (collects metrics and traces inside Borg)
  * Census is the instrumentation for Dapper
  * Going open source -> OpenCensus
  * Avoid ecosystem fragmentation as a tracing vendor
  * Defining data model, not just the API
  * AppDynamics might not be keen on a common ground, the rest views themselves as behind and are interested
  * Long-term, something will equalize
  * Language-specific libraries to send metrics/tracing to any backend
  * Example: Zipkin has many OSS libraries but mostly not maintained by the project; multiple libraries per language
  * Census will have one library per language
  * API for tracing/metrics
  * Provides traces and a dozen metrics out of the box
  * Census does not yet include profiling
  * Any backend supported, contributions welcome
  * Microsoft, Amazon interested
  * Google will be a contributor/partner to the project in the long-term, “not a Google project”
  * Push and pull could be supported; pull can cause contention on hot code paths
  * Zpages agent; executable that talks to Census libraries and hosts a web page serving recent metrics and RPCs
  * For now, inspired by StackDriver data model
    * Views = list of dimensions, metric, aggregation type, window
    * Data (views) currently pushed ‘raw’ to Monarch
 * InfluxDB
 * Open Discussion
 * See GitHub issue tracker for developing consensus.
 * Consensus
   * We spec in Proto
   * Some things might change, v2 vs v3
   * Text format MUST be supported, it’s the lingua franca
     * Easy to write
     * Need it for debugging anyway
    * We will have a test suite to implement against
      * Both linting & validation

## Wednesday 2017-10-11 10:00
 * Going through more issues
 * Fields options:
   * Option A  (“prometheus”, 2 fields):
    * Float64(/Int/Bool)
    * Counter/Gauge/Histogram/Summary
    * Option B (“monarch”, fields):
    * Float/Int/Bool/Distribution
    * Cumulative/Gauge/Delta
  * Option C:
    * Float64/Int/Bool(,Enum?)
    * Cumulative/Gauge
    * Distribution/Scalar/Summary
  * Option D:
     * Float/Int/Bool
     * Counter/Gauge/Counter Histogram/Gauge Histogram
  * Option E:
     * Float/Int/Bool/Float Distribution/Int Distribution
     * Cumulative/Gauge/Delta
  * Option Z - a single field which could be:
     * Float Counter/Cumulative
     * Int Counter/Cumulative
     * Float Gauge
     * Int Gauge
     * Bool Gauge
     * Float Counter/Cumulative Histogram/Distribution
     * Int Counter/Cumulative Histogram/Distribution (sums & boundaries are ints vs floats)
     * Float Gauge Histogram/Distribution
     * Int Gauge Histogram/Distribution
   * Also:
     * Enums
     * Should it be an Enum Gauge Scalar or a Bool Counter Enum?
     * At datamodel level, is it a ‘type’ like int, float or bool?
     * What about float128 or uint64?
     * Geo postitioning
     * Deltas (could ignore for v1, AWS reports a bajillion)
     * Tuples?
       * They get complicated...
     * Do we want to support strings?
     * Will there be OpenMetrics client libraries, or will we just make the Prometheus client libraries OpenMetrics compatible?
   * Start timestamps on samples
     * Allows you to get more accurate rates when samples are sent infrequently 
     * Ie really wide sampling intervals, every hour
     * Can allow you to detect when samples are dropped
     * Can allow you to GC large local histogram dimensions
     * Easily represent deltas in a cumulative stream
     * Just needed for cumulative type
     * Needed for representing deltas (a special case of cumulative).
     * Could be added to a “point” type in the data model, and represented as a second time series for each _child_
     * Is a pretty advanced use case, so client libraries probably doesn’t need to expose it.  But would like to have wire format and data model support it.
   * Exemplars
 * Suggested protos to hash out data structure
   * https://github.com/RichiH/OpenMetrics/tree/master/protos
