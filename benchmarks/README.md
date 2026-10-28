# Comparative benchmarks

This module compares parsing and struct decoding with
[zeebo/bencode](https://github.com/zeebo/bencode/tree/v1.0.0) and
[jackpal/bencode-go](https://github.com/jackpal/bencode-go/tree/v1.0.2).
Dependencies are pinned in [go.mod](go.mod) and [go.sum](go.sum). The local
`replace` measures the current checkout of `github.com/olden/bencode`.
Benchmark dependencies are separate from the library's dependency-free module.

An [example result table](RESULTS.md) includes all measured cases, allocation
counts, environment details, and raw samples. Rerun the suite for your own inputs
and machine before drawing performance conclusions.

## Run

From this directory, run correctness checks first, then collect repeated samples:

```sh
go test ./...
go test -run '^$' -bench . -benchmem -cpu=1 -count=6 -benchtime=500ms > results.txt
```

The output includes `ns/op`, throughput in `MB/s`, allocated bytes (`B/op`), and
allocations per operation (`allocs/op`). Lower time and allocation counts are
better; higher throughput is better. Run without the race detector or coverage
instrumentation when measuring performance.

Use [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) to compare
libraries side by side, or compare the same benchmarks before and after a change:

```sh
go install golang.org/x/perf/cmd/benchstat@latest
benchstat -row .name,/case -col /lib results.txt
benchstat before.txt after.txt
```

For a shorter run focused on torrent metadata:

```sh
go test -run '^$' -bench 'BenchmarkParse/case=SingleFile/' -benchmem -cpu=1
```

Record `go version`, `go env GOOS GOARCH`, CPU model, dependency versions from
`go list -m all`, and the checkout revision with any published results. Run on
an otherwise idle machine and use the same flags for all comparisons. Timing
from shared CI runners is not a stable performance baseline.

## Workloads

All fixtures are deterministic and generated in memory before timing. They use
canonical bencode, unique sorted keys, and integers within `int64`. Binary data
contains NUL and invalid UTF-8 bytes. Torrent fixtures are synthetic metadata,
not real downloads; no files or network requests are needed during benchmarks.

| Case | Encoded size | Content |
| --- | ---: | --- |
| `SmallDict` | 22 B | A name and an integer |
| `IntegerList256` | 1,192 B | Integers from -128 through 127 |
| `Binary64KiB` | 65,542 B | One 64 KiB byte string |
| `NestedList64` | 134 B | A string inside 64 nested lists |
| `SingleFile` | 20,657 B | Metadata with 1,024 synthetic piece hashes |
| `MultiFile64` | 23,719 B | Metadata with 64 file paths and 1,024 piece hashes |

Fixtures are encoded once using the local encoder. Tests check every library's
decoded result against the original expected values, so encoding is neither
timed nor assumed correct merely because our own parser accepts it.

## What is measured

`BenchmarkParse` runs all six cases and returns each library's generic value tree:

| Label | Operation |
| --- | --- |
| `olden` | `NewParser(data)` followed by `Parse()` |
| `olden-copy` | `bytes.Clone(data)` followed by the same parse |
| `zeebo` | `DecodeBytes(data, &value)` with a fresh `any` destination |
| `jackpal` | `Decode(bytes.NewReader(data))` |

The `olden` result borrows byte strings from the input. The `olden-copy` adapter
owns a copy of the complete input, so the caller can reuse its buffer afterward.
Both can keep a whole input buffer alive through a small retained byte string.
The other libraries return strings with independent storage. These differences
are particularly visible in `Binary64KiB`: returning a slice requires much less
work than copying 64 KiB. This is not evidence of equivalent work being faster.

`BenchmarkUnmarshal` runs the two torrent cases into the same Go struct, starting
with a fresh destination on every iteration. For `olden`, it includes both parsing
and `decoder.UnmarshalBDict`; for `zeebo` and `jackpal`, it uses their direct
struct-decoding APIs. All resulting fields are independent of the input buffer.

Parser and reader construction, output allocation, and the explicit input copy
in `olden-copy` are timed. Fixture generation, correctness checks, and comparison
normalization are excluded using `testing.B.Loop`. There is no harness-level
pooling or decoder reuse; internal library optimizations remain enabled.

The libraries run with their default validation settings, which differ. This
suite measures valid-input throughput, not equivalent rejection rules, malformed
input handling, streaming, encoding, `RawInfo` extraction, or retained heap size.
It does not establish that one library is universally faster or safer.

## CI

The main workflow runs this module's correctness tests, `go vet`, and a one-iteration
benchmark smoke check on the same Go version matrix as the library. It does not
enforce timing thresholds. The library's root `go test ./...` excludes this nested
module; use `go test ./...` here to run its tests separately.
