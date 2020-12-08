# Generating the RFC text

The OpenMetrics.md is the source of the RFC text. To generate the output text,
[kramdown-rfc2629](https://github.com/cabo/kramdown-rfc2629) is used. This
requires Ruby.

From the top of the repository:

    bundle install
    bundle exec kdrfc specification/OpenMetrics.md
    mv -v specification/OpenMetrics.txt specification/draft-richih-opsawg-openmetrics-00.txt
