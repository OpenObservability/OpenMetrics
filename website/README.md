# The OpenMetrics website

The website for the OpenMetrics project is available at https://openmetrics.io. The site is built using the [Hugo](https://gohugo.io) static site generator and styled using the [Bulma](https://bulma.io) CSS framework.

## Infrastructure

The site is deployed automatically using [Netlify](https://netlify.com) whenever changes are merged to the `master` branch.

## Setup

In order to build the site, you'll need to have [Yarn](https://yarnpkg.com) and [Hugo](https://gohugo.io) (version 0.46 or later) installed and the OpenMetrics repo cloned locally:

```bash
git clone https://github.com/OpenObservability/OpenMetrics
cd OpenMetrics/website
```

## Building

In order to build the site, first install Bulma using Yarn:

```bash
yarn
```

Then you can build the site using Hugo:

```bash
hugo
```

## Running the site locally

To run the site in development mode:

```bash
hugo server
```

Then open your browser to http://localhost:1313. Whenever you make changes to the site's assets, Hugo will trigger auto-reload in the browser.
