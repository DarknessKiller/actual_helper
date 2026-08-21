# Provider config: in-memory cache (evidence)

v0.3.1 had no benchmarks. The per-request hot path
(`services.reloadProvider` -> `config.Loader.ProviderConfig`) used to do
`os.Stat` + `os.ReadFile` + `json.Unmarshal` on **every** `ConvertFile`
request. The config-lifecycle work (`POST /config`) stores the user-supplied
config in memory; `ProviderConfig` now reads memory under the existing mutex
with no per-request file I/O.

## Benchmarks added

- `internal/config/config_bench_test.go` — `BenchmarkLoaderProviderConfig`
  with three sub-benchmarks: `empty` (no config loaded), `loaded`
  (in-memory config), and `from_file` (synthetic pre-cache path that repeats
  `os.Stat` + `os.ReadFile` + `json.Unmarshal` per call, modelling the old
  behaviour so the win is measurable).
- `internal/services/services_bench_test.go` — `BenchmarkToActualCSV` on a
  fixed 100-row slice.
- `internal/providers/tng/tng_bench_test.go` — `BenchmarkTNGParsePDFText`
  (TNG is PDF-only; `ParseCSV` returns "not supported", so the real TNG
  parse path is `ParsePDFText`). Per-call `slog.Info` logging is silenced so
  the bench measures parse cost, not log I/O.

Run with:

```
CGO_ENABLED=0 go test -bench=. -benchmem -run='^$' \
  ./internal/config/... ./internal/services/... ./internal/providers/tng/...
```

## Measured result (go1.27.0, CGO_ENABLED=0, Ryzen 7 PRO 6850H)

```
BenchmarkLoaderProviderConfig/empty-4     190975800   6.359 ns/op    0 B/op   0 allocs/op
BenchmarkLoaderProviderConfig/loaded-4     6480392   182.2  ns/op  256 B/op  5 allocs/op
BenchmarkLoaderProviderConfig/from_file-4     79376  14980   ns/op 2932 B/op 26 allocs/op
BenchmarkToActualCSV-4                        46039  25491   ns/op 30976 B/op 105 allocs/op
BenchmarkTNGParsePDFText/empty-4              91582  12929   ns/op  3668 B/op  59 allocs/op
BenchmarkTNGParsePDFText/loaded-4            90184  12902   ns/op  3675 B/op  59 allocs/op
```

## The win

`ProviderConfig` with a loaded in-memory config is **~82x faster** than the
pre-cache per-request file-read path (14980 ns -> 182 ns) and drops
**26 -> 5 allocs/op**. With no config loaded (the default until a user loads
one) it is **0 allocs / ~6 ns** — a single guarded memory read.

The remaining 5 allocs in the `loaded` case are the inherent global +
per-provider slice merge (`append` into fresh slices); precomputing a merged
config per provider name would add a speculative data structure for no
measurable benefit, so the simple read-merge path is kept.

`ToActualCSV` and `TNGParsePDFText` baselines are recorded for future
regression tracking; no change was needed there.
