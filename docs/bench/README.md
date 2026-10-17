# Mail driver benchmarks

Run from the repository root with Go 1.24 or later (use a current patch release), Node.js 24, and Docker:

```sh
make bench
make bench-check
```

The suite tests all eight drivers through `mail.New(driver).Send`. Every message has one recipient, text and HTML bodies, and a 4 KiB attachment. Seven one-second samples per driver run serially with `GOMAXPROCS=1`. Message construction, fixture startup, driver construction, and connection warmup are outside the timer.

API drivers use local HTTP acceptance fixtures in separate child processes. Requests go over reused plaintext loopback connections; SES includes SDK signing. Server allocations do not enter client B/op or allocs/op measurements. These fixtures consume payloads and return provider-shaped acknowledgements. They do not emulate provider-side validation, throttling, queues, TLS, the public network, or inbox delivery.

SMTP sends to a disposable Mailpit container, bound to a random loopback port. The runner removes only its own container on completion. Log serializes JSON with bodies to `io.Discard`. Fake records messages and resets every 128 sends; the amortized reset cost is included. These local drivers have different semantics from network senders, so their timings are not interchangeable delivery guarantees.

The module uses relative replacements for the root and SES modules so benchmarks exercise repository source, including unpublished sibling changes. It is tooling and receives no release tags. Production modules keep their existing dependency and Go-version boundaries.

`benchmarks.txt` preserves Go's output. `benchmarks_rows.json` preserves all 56 samples, host and Go versions, the image ID, and fingerprints of the harness, manifests, and measured library sources. `make bench-check` detects stale measurements and verifies that README medians match the raw data. CI checks artifacts and tests the fixture module; it does not refresh performance numbers on shared runners.

The GoForj performance page consumes the committed JSON artifact. Regenerate the artifact and README together when changing measured library code.
