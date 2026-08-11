# E2 controlled browser comparison

**Status:** Reproducible controlled experiment; no universal comparison claim

**Evidence date:** 2026-08-10

This diagnostic compares Chromium 150.0.7871.114 rendering one controlled
156-byte JSON fixture with the promoted, offline typed adapter for that same
fixture. It is outside the trusted runtime. The browser subprocess uses a
fresh profile, disables background features, and maps every non-loopback host
to `NOTFOUND`. Chromium still attempted built-in requests to three Google
hosts; the resolver rule blocked them. The typed adapter performed no network
request.

Command:

```bash
scripts/compare-e2-browser.py
```

Observed result:

| Measure | Controlled browser | Promoted typed adapter |
|---|---:|---:|
| Wall time | 2.586759 s | 0.062731 s |
| Peak child resident memory | 240,764 KiB | 22,908 KiB |
| Agent-input representation | 320 bytes of dumped DOM | 191 bytes of compact typed values |
| Local representation bytes read | 156 | 156 from committed replay evidence |
| Network requests | 1 allowed loopback plus blocked browser background attempts | 0 |
| Evidence-bearing fields | 0 | 5 |
| Full typed result including proof references | not applicable | 3,674 bytes |

The typed result identifier was
`sha256:8bafb410dee23a3e6a5011f81b46531083d82c9395e0b2e9d134b702433ee972`.
The full result is larger than the dumped source because it intentionally
includes native and semantic views, transformations, mappings, and digest
bindings. The 191-byte compact value view is shown only to describe the agent
input shape; the complete 3,674-byte response remains available for audit.

This one run observed about 41 times lower wall time and 10.5 times lower peak
resident memory for the admitted adapter. Those ratios are not generalized:
they depend on browser startup, the tiny local fixture, cache state, host,
operation, and measurement method. The experiment says nothing about
arbitrary HTML pages or browser discovery and is not evidence that TWIRX is
always faster than a browser.
