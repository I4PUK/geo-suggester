### geo-suggest

REST API for geo suggestions in hotel search.

#### Configuration

Copy `.env.example` to `.env` and set values for your environment. Required variables are listed in `.env.example` (Elasticsearch, Redis, gRPC endpoints, optional Sentry DSN).

#### Known limitation

Some Go dependencies still use private module paths (geo-client, suggest-core, internal golang libraries, contracts). They are not available publicly, so a full public build needs those packages replaced or published separately.
