# bencode

[![CI](https://github.com/olden/bencode/actions/workflows/ci.yml/badge.svg)](https://github.com/olden/bencode/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/olden/bencode.svg)](https://pkg.go.dev/github.com/olden/bencode)

A Go library for parsing and encoding [bencode](https://www.bittorrent.org/beps/bep_0003.html),
the serialization format used by BitTorrent. It provides strict parsing, canonical
encoding, dictionary-to-struct decoding, and access to the original `info` bytes
for BitTorrent v1 info hashes.

Requires **Go 1.24.1 or later**. Uses only the Go standard library.

## Installation

```sh
go get github.com/olden/bencode@latest
```

| Package | API |
| --- | --- |
| `github.com/olden/bencode` | `NewParser`, `Parser.Parse`, value types, and `Parser.RawInfo` |
| `github.com/olden/bencode/encoder` | `Encode`: Go values to bencode |
| `github.com/olden/bencode/decoder` | `UnmarshalBDict`: a parsed dictionary to a Go struct |

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/olden/bencode"
	"github.com/olden/bencode/decoder"
	"github.com/olden/bencode/encoder"
)

func main() {
	p, err := bencode.NewParser([]byte("d4:name3:cat4:sizei7ee"))
	if err != nil {
		log.Fatal(err)
	}
	value, err := p.Parse()
	if err != nil {
		log.Fatal(err)
	}
	dict, ok := value.(bencode.BDict)
	if !ok {
		log.Fatal("expected dictionary")
	}

	var out struct {
		Name string `bencode:"name"`
		Size int    `bencode:"size"`
	}
	if err := decoder.UnmarshalBDict(dict, &out); err != nil {
		log.Fatal(err)
	}

	encoded, err := encoder.Encode(out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Name, out.Size)
	fmt.Println(string(encoded))
}
```

Output:

```text
cat 7
d4:name3:cat4:sizei7ee
```

Pass the result of `Parse` directly to `encoder.Encode` to re-encode a value
without mapping it to a struct. See [example_test.go](example_test.go) for runnable
examples.

## Parsing and value types

`Parse` returns one of these types through `BValue`:

| Bencode type | Go type | Underlying type |
| --- | --- | --- |
| Integer | `BInt` | `int64` |
| Byte string | `BString` | `[]byte` |
| List | `BList` | `[]BValue` |
| Dictionary | `BDict` | `map[string]BValue` |

Each parser is single-use and accepts exactly one complete document. It rejects
malformed numbers, integer overflow, negative string lengths, leading zeros,
negative zero, truncated input, and trailing data. Dictionary keys must be unique
and sorted by raw bytes. Byte strings may contain any bytes, including invalid
UTF-8 and NUL bytes.

### Limits and ownership

- Integers are limited to the `int64` range.
- Lists and dictionaries may nest up to `MaxDepth` (128); the root container
  counts as depth 1. Encoding and decoding enforce the same container limit.
- Parsed `BString` values and `RawInfo` reference the input buffer. Keep it
  unchanged while using them, or copy the bytes with `bytes.Clone`.
  Retaining a small byte string can keep the entire input buffer alive; copy
  individual fields when storing them long-term or before reusing a pooled buffer.
- The library does not impose a total input-size limit. Bound untrusted input
  before reading it into memory.
- Parsing validates bencode syntax, not required torrent fields or metadata
  consistency. Applications must validate their own schema.

## Encoding and struct decoding

`encoder.Encode` supports signed integers and unsigned integers within the
`int64` range, strings, byte slices and arrays, other slices and arrays as lists,
maps with string keys, and structs. Named types and non-nil pointers and
interfaces are supported. Dictionary keys are sorted by raw bytes.

Nil maps and slices encode as empty dictionaries, lists, or byte strings.
Nil pointers and interfaces, booleans, floating-point values, and other
unsupported types return errors. No partial output is returned on failure.
Container and pointer-chain limits prevent infinite recursion through cycles.

`decoder.UnmarshalBDict` requires a non-nil pointer to a struct. It supports
nested structs, pointers, typed collections, and empty interfaces. Incompatible
types and integer overflow return errors identifying the field or collection
element. Unknown keys are ignored, and absent fields keep their existing values.
Decoded bytes and collections are copied; the destination remains unchanged on
error.

Both packages use `bencode` struct tags:

```go
type Entry struct {
	Name    string `bencode:"name"`
	Length  int64  `bencode:"length"`
	Ignored string `bencode:"-"`
}
```

Untagged fields use their Go field names. Unexported fields are ignored.
Duplicate field names and tag options, including `omitempty`, are rejected.

## BitTorrent v1 info hashes

Set `p.RawInfo = &rawInfo`, where `rawInfo` is a `[]byte`, before calling `Parse`.
After a successful parse, verify that the root is a `BDict` and its `info` value
is also a `BDict`, then calculate `sha1.Sum(rawInfo)` using `crypto/sha1`.
[ExampleParser](example_test.go) demonstrates these steps.

Only the root dictionary's `info` value is captured; nested `info` keys do not
replace it. The output is empty when the key is absent or parsing fails.
Hash the original bytes: converting metadata to a struct and re-encoding it
can add or omit fields and change the hash.

## Development

Run these commands from the repository root:

```sh
go test -race -count=1 -cover ./...
go vet ./...
gofmt -l .
```

`gofmt -l .` should produce no output; use `gofmt -w .` to format files.
[GitHub Actions](.github/workflows/ci.yml) runs tests with the race detector and
coverage, `go vet`, and a formatting check on Go **1.24.x, 1.25.x, 1.26.x, and
1.27.x** for pushes and pull requests. Each job uses its selected toolchain
without automatic upgrades.

Run benchmarks or additional fuzzing locally:

```sh
go test -run '^$' -bench . ./...
go test -run '^$' -fuzz '^FuzzParser_NoPanic$' -fuzztime=30s
go test -run '^$' -fuzz '^FuzzParseEncodeRoundTrip$' -fuzztime=30s
go test -run '^$' -fuzz '^FuzzUnmarshalRoundTrip$' -fuzztime=30s
```

Regular tests include the fuzz seed corpus. Fuzz targets cap input at 64 KiB;
this is a test-harness limit, not a library limit. Additional scenario sources
are documented in [parser_reference_test.go](parser_reference_test.go).

## Comparative benchmarks

The separate [benchmarks module](benchmarks/README.md) compares this library with
`zeebo/bencode` and `jackpal/bencode-go` on identical inputs. It measures generic
parsing and full decoding into structs, reporting time, throughput, and allocations.
It also measures parsing after copying the input to show the cost of independent
buffer ownership. Third-party dependencies are confined to that module.

```sh
cd benchmarks
go test ./...
go test -run '^$' -bench . -benchmem -cpu=1 -count=6 -benchtime=500ms
```

See the benchmark README for pinned versions, workloads, ownership differences,
and commands to produce comparison tables with `benchstat`. CI checks benchmark
correctness on the same Go version matrix, without performance thresholds.

## Contributing

Keep changes focused, format Go code with `gofmt`, and add regression tests for
behavior changes. Run the development checks before opening a pull request.
Describe the problem, the resulting behavior, and how you tested it. For bug
reports, include a minimal reproducer, your Go version, and the expected result.
