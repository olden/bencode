# Example benchmark results

Measured on 2026-10-28 with **Go 1.27.3, macOS arm64, Apple M1 Pro**,
`GOMAXPROCS=1`. The local library was an unreleased working tree;
comparison versions were `zeebo/bencode v1.0.0` and `jackpal/bencode-go v1.0.2`.

These are medians of six samples per benchmark, with 500 ms per sample.
No race detector or coverage instrumentation was enabled. Values are a snapshot,
not a performance guarantee or a claim of statistical significance.

```sh
go test -run '^$' -bench . -benchmem -cpu=1 -count=6 -benchtime=500ms
```

[Raw samples](results/2026-10-28-darwin-arm64.txt) include throughput and all
individual measurements. See the [methodology](README.md) before interpreting
the tables. `olden` borrows input bytes; `olden-copy` copies the entire input
inside the timed operation. The other parsers return independently stored strings.
Struct decoding produces independent values for all three libraries.

## Generic parsing

| Case | Library | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `SmallDict` | `olden` | 378.8 | 512 | 7 |
| `SmallDict` | `olden-copy` | 423.1 | 536 | 8 |
| `SmallDict` | `zeebo` | 1,470.5 | 4,728 | 23 |
| `SmallDict` | `jackpal` | 474.9 | 456 | 12 |
| `IntegerList256` | `olden` | 14,143.0 | 10,520 | 140 |
| `IntegerList256` | `olden-copy` | 14,590.0 | 11,800 | 141 |
| `IntegerList256` | `zeebo` | 28,997.0 | 22,744 | 414 |
| `IntegerList256` | `jackpal` | 11,895.5 | 10,424 | 139 |
| `Binary64KiB` | `olden` | 119.9 | 168 | 3 |
| `Binary64KiB` | `olden-copy` | 6,285.0 | 73,896 | 4 |
| `Binary64KiB` | `zeebo` | 15,033.5 | 135,352 | 8 |
| `Binary64KiB` | `jackpal` | 13,242.0 | 131,142 | 4 |
| `NestedList64` | `olden` | 4,358.5 | 2,728 | 131 |
| `NestedList64` | `olden-copy` | 4,466.5 | 2,872 | 132 |
| `NestedList64` | `zeebo` | 13,327.5 | 12,988 | 264 |
| `NestedList64` | `jackpal` | 3,866.5 | 2,632 | 132 |
| `SingleFile` | `olden` | 1,064.0 | 992 | 19 |
| `SingleFile` | `olden-copy` | 3,179.5 | 22,752 | 20 |
| `SingleFile` | `zeebo` | 8,066.5 | 46,608 | 67 |
| `SingleFile` | `jackpal` | 6,166.0 | 42,178 | 40 |
| `MultiFile64` | `olden` | 36,011.0 | 33,880 | 666 |
| `MultiFile64` | `olden-copy` | 38,142.5 | 58,456 | 667 |
| `MultiFile64` | `zeebo` | 82,663.5 | 95,416 | 1,877 |
| `MultiFile64` | `jackpal` | 48,413.0 | 79,164 | 1,199 |

## Decoding into structs

| Case | Library | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `SingleFile` | `olden` | 4,336.5 | 21,792 | 25 |
| `SingleFile` | `zeebo` | 9,133.0 | 46,064 | 54 |
| `SingleFile` | `jackpal` | 8,115.0 | 42,033 | 35 |
| `MultiFile64` | `olden` | 60,557.5 | 61,616 | 930 |
| `MultiFile64` | `zeebo` | 85,268.5 | 74,944 | 1,479 |
| `MultiFile64` | `jackpal` | 62,472.5 | 84,740 | 810 |

`Binary64KiB` highlights buffer ownership, not a faster byte-copy operation:
the borrowed parser reads the length prefix and returns a slice without scanning
or copying the payload. Its reported MB/s is input size divided by time, not
physical memory bandwidth. Allocation metrics do not measure how much of an
input buffer remains reachable after retaining a field.

Performance varies by workload. In this run, `jackpal` had lower median parsing
time on the integer-list and nested-list cases. Repeat measurements on your
hardware and representative inputs before choosing or optimizing an API.
