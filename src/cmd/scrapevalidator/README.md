# scrapevalidator

A command tool to scrape an OpenMetrics endpoint and validate against the OpenMetrics spec.

## Compile

From the /src directory:

```
make scrapevalidator
```

## Example

Here are some examples of running the tool from the root directory.

```
./bin/scrapevalidator --endpoint "http://localhost:9100/metrics"
2021/06/15 16:23:32 scraped successfully
2021/06/15 16:23:32 parsed 10 data points, validated successfully
```