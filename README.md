# apps

monorepo of golang apps for fun and profit

### `image-builder`

`image-builder` builds images inside of a Nomad cluster via HTTP API calls.

### `nomad-event-stream`

`nomad-event-stream` streams all of Nomad's events into a Redis stream for other clients to consume.

### `nomad-external-dns`

`nomad-external-dns` is a service that watches for changes in Nomad's service registry and updates DNS records via AXFR/IXFR/native dynamic DNS update.

### `nomad-service-cleaner`

`nomad-service-cleaner` is a one-shot program that cleans p defunct Nomad service registry entries.