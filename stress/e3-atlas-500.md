# E3 Atlas-500 runtime workload

Run the bounded local workload with:

```bash
make stress-e3-500
```

The default run first derives the admission work state for every selected
origin, then starts the real Atlas HTTP service on literal loopback, walks
all five 100-record pages, directly describes every selected origin, verifies
all family and orthogonal-state filters, validates all 500 dry-run frontier
outcomes, rejects eight malformed or unauthorized requests, and performs 100
complete concurrent lookup rounds (50,000 requests). It then restarts the
service and requires the status artifact to remain byte-identical. The queue
reports prepared and unprepared dossiers independently; a missing dossier is
never treated as catalog or policy approval.

`TW_ATLAS_STRESS_ROUNDS` is bounded to 1–1000 and
`TW_ATLAS_STRESS_WORKERS` to 1–128. The client accepts only an exact HTTP
origin whose host is a literal loopback address and whose port is explicit.
It cannot retrieve a selected public origin.

Passing this workload proves that the real read-only control plane handles all
500 selected origins under concurrency and produces an explicit frontier
decision for each one. It does not promote policy, technical, publisher,
health, adapter-trust, or mapping-trust state, and it does not claim that 500
origins are live.
