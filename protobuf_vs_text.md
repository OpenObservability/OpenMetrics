# Prometheus Exposition Format: Protobuf vs. Text

_by Björn “Beorn” Rabenstein, SoundCloud Ltd._ [@beorn](https://github.com/beorn7)

Prometheus 1.x supports two exposition formats:
- a Protocol Buffer format (“protobuf”).
- a line-oriented text format (”text”).

Evolving the Prometheus exposition format(s) into a standard raises the
question if both formats or only one of them should be part of the
standard. Currently, the community gravitates towards the text format for a
number of reasons. However, a number of aspects haven't been discussed widely
so far. This document intends to raise awareness for them and turn the format
decision into an informed one, whatever it will ultimately be.

_Disclaimer: The author of this document joined the project in late 2013. While
that's fairly early, he wasn't part of the project from the beginning. The
claims about the “original vision” and similar as stated below are therefore
not necessarily accurate but merely to the best of the author's knowledge. In
the nascent project, a lot of knowledge was “tribal” and not yet documented in
written form. Furthermore, the author was not involved in the recent
development of Prometheus 2, which might lead to incorrect statements
below. Those, at least, can be fixed easily. Corrections are welcome. On the
other hand, the author was the lead designer of the text format. Thus,
statements about original design decisions for the text format can be
considered canonical._

Note about terminology: The terms _histogram_ and _summary_ in italics refer
specifically to the Prometheus metric types. Their meaning has grown
historically and is not necessarily in line with the general understanding of
those terms. Counters and gauges, in contrast, are fairly well defined and no
typographical distinction between the general terms and the Prometheus metric
types is required.

## Historical notes

### Original role of the protobuf and text format

_Note: Prometheus once also supported a JSON exposition format. The reasons for
dropping it are
[discussed elsewhere](https://youtu.be/4DzoajMs4DM?t=11m59s). While interesting
for the complete picture, they are not important for the discussion of protobuf
vs. text format._

The protobuf format has a data model that is fundamentally richer than what
Prometheus supports (even today with Prometheus 2.x):
- It contains doc strings (`help`) per metric family.
- It has first-class support for different metric types.
- It has first-class support for complex metric types that cannot be
  represented with a single floating point number as a value:
  - Bucketed _histograms_.
  - _Summaries_ with pre-calculated quantiles.

Prometheus throws away doc strings and the metric type and maps _histograms_
and _summaries_ into a number of simple gauge- or counter-like metrics (one per
bucket of a _histogram_, one per quantile of a _summary_, one for the number of
observations, one for the sum of observations).

The vision behind that was to create a rich “future-proof” data model, which
Prometheus would later “grow into”. In that way, we could create a working
prototype for Prometheus, covering the simple cases and harvesting the lowest
hanging fruit, while instrumentation libraries can already support the rich
data model, to the benefit of future Prometheus version or even non-Prometheus
consumers.

The original motivation for the text format was twofold:
- There are plenty of situations where the machinery of creating the protobuf
  wire format is not readily available, with the proverbial example being a
  shell script that needs to push a small number of samples to the Pushgateway,
  or a minimal Python program that intends to serve a small number of metrics via the
  HTTP server provided by the standard library. We call that use-case _ad-hoc
  exposition_ or _library-less exposition_.
- The data model of the protobuf format was often perceived as needlessly
  complicated, on the one hand because the Prometheus server was throwing away
  most of the hard work done before, on the other hand because a simple
  use-case only consisting of gauges and counters doesn't benefit much from the
  richness of the data model.

This resulted in two design principles for the text format that are very
important to keep in mind:
- The text format should be _easy and fault-resistant to assemble_, based on
  the assumption that it is, in most cases, either directly assembled by a
  human, or assembled programmatically but without much library support,
  i.e. conforming to strict format specifications would be a relevant
  burden. Minimizing efforts for the producer outweighs the efforts for the
  consumer.
- The text format should work well for easy cases (a few gauges or counters)
  but doesn't have to deal well with complex cases (mostly _histograms_ and
  _summaries_).

These design guidelines explain properties of the text format that are
nowadays considered problematic:
- The liberal use of whitespace everywhere.
- The verbose and fragile representation of _histograms_ and _summaries_. (In
  fact, the original code of the text format parser stated that it is in
  general a bad idea to represent _histograms_ and _summaries_ with the text
  format.)

Plenty of concerns were raised back then about the performance penalty of
parsing text vs. decoding protobufs and the loss of robustness, consistency,
and stringency. The way of breaking complex metric types into simple metric
lines was the same as done by the Prometheus server to save the “rich”
_histograms_ and _summaries_ in its “poor” embedded TSDB. This was explicitly
meant as convenient coincidence and not as a leak of Prometheus implementation
details into the exposition format. To make sure of that, the text format
parser was implemented to convert the text format into protobuf (rather than
the interval data model of the Prometheus server), so that we could still
follow our vision of growing the internal data representation into something
closer to the rich data model of the protobuf format.

The text format was also designed to closely match the appearence of metric
selectors in PromQL.

The design efforts paid off quickly as the text format became popular quite
quickly, with some intended and some not so intended side effects:
- The text format was used as the usual human-readable representation of
  metrics when looking at the `/metrics` endpoint with a browser. (An easy to
  implement alternative would have been to show the text representation of the
  protobuf format. But that would have been quite hard to read for humans,
  especially in simple cases.)
- It enabled the textfile collector in the
  [Node Exporter](https://github.com/prometheus/node_exporter), one of the most
  convenient ways of exposing metrics: Drop a snippet with metrics in the text
  format into a designated directory, and the Node Exporter will immediately
  start to expose it. Many questions of “How to export X?” could now be
  answered by one line of Bash run by cron.
- The maintainer of the Python and Java client libraries came to the conclusion
  that a dependency on protobuf causes issues in those languages and would
  impede adoption. Thus, he happily implemented those libraries without
  protobuf support. (In case of the Java library, his rewrite had no protobuf
  support while the original, quite prototypical implementation still had it.)
  Other libraries mostly followed that example, with the notable exception of
  the Go client library (which is also the oldest library). The Go library
  follows the spirit of the original vision of approaching the protobuf data
  model to the fullest possible extent: It internally directly acts on protobuf
  messages. Only if explicitly asked for the text format (e.g. via an HTTP
  request by a browser), it converts the protobuf representation into the text
  format.

The performance overhead of the text format was in general considered as small
enough to not worry about client libraries without protobuf support.

### Recent developments

For the upcoming Prometheus 2, the internal storage was completely rewritten,
with dramatic performance improvements along almost all dimensions. The rewrite
went along with a new internal data model for metrics. However, this didn't
follow the original vision of approaching the rich protobuf data model. On the
contrary, it was in embracing the text format instead, with various beneficial
results.

An important incentive was performance analysis. As it turned out, the whole
decoding chain from the protobuf wire-format into the internal data model of
Prometheus was quite wasteful. In Prometheus 1.x, starting the chain at the
text format was unsurprisingly even more wasteful because the text parser was
parsing the text format into protobufs first, as described above.

For Prometheus 2, an allocation-free text-format decoder was written that
decodes the text format directly into the (new and improved) internal data
model. As it turned out, this decoding chain is dramatically more efficient
than the original protobuf chain. Not without historical irony, things changed
completely, from “You can use the text format, but only the protobuf format
will give you really high performance.” to “You should avoid the protobuf
format, it eats quite a few of your cores.”

On top of that, Prometheus 2 avoids calculating a metrics hash on each
ingestion. This tweak also leverages how the text format is laid out on the
wire. In high-ingestion scenarios, metrics hashing would otherwise have become
the dominant CPU consumer.

## Comparison of the protobuf and text format

This section compares different aspects and implications of both formats.

### General applicability and upgrade paths

It is important to keep in mind that protobuf encoding and decoding is not
necessarily fast (as was naively assumed at the time the text format was
created and concerns were raised that the text format decoding would be too
slow). A hand-written encoder/decoder for a custom format might easily
outperform the code auto-generated by the protobuf compiler. (But see the
[section about performance below](#decoding-and-encoding-performance) for
further considerations.) Arguably, the key advantage of protocol buffers is to
provide the following featurs _at a decent encoding/decoding performance_:
1. Create encoders and decoders in all supported languages from a single
   `.proto` file.
2. Changes of the format are therefore easy to push (the `.proto` file is the
   only code to change manually). Plus, protocol buffers are designed to allow
   forward and backward compatibility.
3. The structured data as described by the `.proto` file can be directly
   operated on in the target language. There is no need to translate to and
   from the internal data model. The protobuf description _is_ the data model.

One might argue that each of the items above are of limited relevance for
Prometheus:
1. Encoders for both the protobuf and text format exist in various languages
   already (in the form of Prometheus client libraries). For some languages,
   there are even decoders available. (However, either direction is not at par
   yet with the languages for which protobuf compilers exist).
2. With the maturity of the Prometheus ecosystem, future changes of the
   exposition format are expected to be rare. (This expectation might be seen
   as naive, though.)
3. Prometheus so far has failed to benefit from this item, as the internal data
   model is different from the protobuf description in both Prometheus 1.x
   and 2.x (which is arguably eiter a shortcoming of Prometheus or of the
   protobuf representation in Go, see also the
   [section about the data model below](#implied-data-model)).

The arguments above are based on a “Prometheus only” scenario. Obviously, the
whole point of evolving the exposition format into a standard is applicability
outside of the Prometheus ecosystem, too. Viewing each item from that
perspective changes the outcome considerably:
1. The ability to easily create encoders and in particular decoders in many
   languages, as provided by the protobuf format, becomes more important.
2. As the standardization will add more requirements for the exposition format
   (e.g. timestamps of sub-millisecond precision, different value types, …),
   more changes are expected in the process.
3. Encoders and decoders created for Prometheus encode from and decode to the
   internal Prometheus data model. This might not be the data model
   non-Prometheus users want to use. See the
   [section about the data model below](#implied-data-model) for details.

In summary, the advantages of a protobuf format are more relevant for a widely
adopted standard than for a format meant to be used predominantly with the
Prometheus ecosystem.

### Implied data model

As described above, the internal Prometheus data model is different from the
data model of the protobuf format. The original reason was that Prometheus had
to reach the state of a working prototype, and more complex metric
representation was postponed. However, there is also a problem with the code
the official Go protobuf compiler generates. It's just not very idiomatic for
Go, so acting on the generated data structures as the internal data model is
cumbersome (as demonstrated in the Prometheus Go client library). This has
somewhat improved with proto3, but Prometheus hasn't moved to proto3 yet, see
[this GH issue](https://github.com/prometheus/docs/issues/549) for
details. Another option would be an alternative Go protobuf implementation
like [gogo/protobuf](https://github.com/gogo/protobuf), which arguably
generates more idiomatic Go code and would make it easier to act directly on
the protobuf data structures.

For the reasons stated above, the text format follows the internal Prometheus
data model much more closely than the protobuf format. While explicitly not
intended as such, the text format effectively leaks the internal Prometheus
model into the exposition format. The main implication is that complex metric
types (_histogram_, _summary_) are broken down into individual metric lines
with 64bit floating point numbers as data type and some “magic” labels (`le`,
`quantile`). This is verbose and prone to inconsistencies. For example, the
text format legally allows to specify a timestamp per line. In a _histogram_
and _summary_, however, all quantiles, buckets, and the sum and count of
observations must have the same timestamp. Furthermore, data types are not
appropriate anymore. The count of a bucket and the count of observations are
supposed to be unsigned integers but are represented as floating point numbers
in the text format. The value of the `le` label is supposed to be a floating
point number but represented as an UTF-8 string. The protobuf format doesn not
have any of these issues.

A different aspect is that the representation of histograms as naive buckets is
most certainly not the end of the journey. Many ideas about dynamic bucketing
and/or inexpensive representation of higher resolution are floating around,
which would result in representing some kind of digest or even binary
compressed data (more research needed, but this is definitely a very hot topic
with huge potential for future uses). The text format in its current form
needed to change fundamentally to accommodate such innovations.

Highly relevant here are the
[comments by Sumeer Bhola and Jeromy Carriere](https://docs.google.com/a/soundcloud.com/document/d/1TapRdI5Tu2e8AIqREtdv_C36t4GGsDQr64KixE-gWu8/edit?disco=AAAABLb2KLE). An
important conclusion is that flattening of a complex metric type is not the
responsibility of the exposition format but that of the consuming backend.

It should also be noted here that the current way how Prometheus handles
complex metric types has a lot of issues even with a completely
Prometheus-centric view. See
[this slide deck from the Prometheus developer summit Berlin 2017](https://docs.google.com/presentation/d/1WUSerlBAB6I6jz6Vo3d6VScCgMk7eOrTCOCJZIcjnyc/edit?usp=sharing)
for details. Since a change within Prometheus is highly desirable. We could,
ironically, end up in a state where the standardized text format would still
follow the leaked internal data model, even after the internal data model of
Prometheus has improved.

There is the option of changing the text format to not break down complex
metric type. A histogram sample could look like this:
```
some_histogram{foo="bar"} {sum: 47.11, count: 42, 0.1: 1, 0.2: 5, 0.5: 12, 1: 37}
```
This would solve many of the issues above, but would still make it quite hard
to change to advanced representations of histograms. Lines could also become
very long with many buckets. See also the
[section about performance](#decoding-and-encoding-performance).

### Verbosity and compression

The text format is needlessly verbose for _histograms_ and _summaries_. The
impact is particularly high in a use-case with many buckets on a highly
dimensional metric. Since the data is mostl redundant, it compresses nicely. In
the Prometheus context, scrapes are gzip-compressed by default, which results
in no noticable size difference between text and protobuf scrapes. However,
there are scenarios where the resources needed for compression might be a
relevant burden. The most likely case is a resource-tight monitoring
target. But with the dramatic improvements in ingestion performance featured by
Prometheus 2.x, compression might become an issue on the side of the Prometheus
server, too (or other consumers, as we are talking about making the exposition
format generally useful here, not just for Prometheus).

A change of the text format as suggested in the previous section, where buckets
of a _histogram_ and quantiles of a _summary_ are all listed on the same line,
would help here. But see also the next
[section about performance](#decoding-and-encoding-performance).

### Ingestion performance and the benefits of a binary encoding

As reported above, the new hand-coded text format decoder in Prometheus 2.x is
dramatically more efficient than the protobuf decoding in Prometheus 1.x. Also,
the text format was leveraged for ingestion tweaks, saving time on hashing
metrics and labels. This result deserves a closer look.

For one, there is the aspect of a hand-coded vs. a generated decoder. If
maximum performance is the first priority, a protobuf decoder could be
hand-coded, too. The _necessity_ of hand-coding a decoder for the text format
should not be taken as an _advantage_ for the text format. While hand-coding a
protobuf decoder would be quite extreme, the ingestion tweaks implemented on
top of the on-the-wire representation of the text format could fundamentally
implemented in the same way on top of the protobuf wire format.

Then there are the known problem with the official Go protobuf compiler. Not
only is the generated code not very idiomatic, as discussed above, it is also
quite inefficient. Alternative protobuf implementation perform way better,
notably [gogo/protobuf](https://github.com/gogo/protobuf). See
[this CloudNativeCon talk](https://youtu.be/Bmzx-5uExPM?t=15m57s) for a case
study of protobuf decoding performance of the official protobuf compiler vs
[gogo/protobuf](https://github.com/gogo/protobuf).

Finally, the current payload is mostly strings, which are naturally encoded in
more or less the same way in a text-based format vs. protobuf. The encoding of
numbers is fundamentally less efficient in a text-based format. One of the pain
points with Prometheus right now is the relatively high cost of a histogram
bucket. Users have to be very judicious with their bucketing schemes. With an
improved Prometheus (or with another backend that deals differently with
histograms), we will see payloads dominated by histogram buckets – or
by more efficient representations thereof, i.e. some kind of digest or binary
compressed data. In either case, it will tip the balance from string-dominated
towards mostly numeric or even binary data, for which a text-based formet is
not a natural fit.

### Required changes for standardization

The above suggests that the move from a Prometheus-targeted format to a
generally useful metrics exposition format will require many changes of the
text format. However, it should be noted that also the protobuf format will
require some changes, too. This document is not meant to imply that all
concerns will go away by choosing the protobuf format. However, there are fewer
fundamental problems, and the protobuf format is better suited for ongoing
improvements and additions of the format.

### Implications for gRPC

With the rise of [gRPC](http://www.grpc.io/), the request for exposing metrics
via gRPC was inevitable. While gRPC is in principle payload-agnostic, it
integrates most naturally with protocol buffers. We would essentially send
structured data as an opaque strings via gRPC if we used the text format.
