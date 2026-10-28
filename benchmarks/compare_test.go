package benchmarks

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	jackpal "github.com/jackpal/bencode-go"
	olden "github.com/olden/bencode"
	"github.com/olden/bencode/decoder"
	zeebo "github.com/zeebo/bencode"
)

type parserImplementation struct {
	name  string
	parse func([]byte) (any, error)
}

var parsers = []parserImplementation{
	{"olden", parseOlden},
	{"olden-copy", func(data []byte) (any, error) { return parseOlden(bytes.Clone(data)) }},
	{"zeebo", func(data []byte) (any, error) {
		var value any
		err := zeebo.DecodeBytes(data, &value)
		return value, err
	}},
	{"jackpal", func(data []byte) (any, error) { return jackpal.Decode(bytes.NewReader(data)) }},
}

var unmarshallers = []struct {
	name      string
	unmarshal func([]byte, *torrent) error
}{
	{"olden", func(data []byte, out *torrent) error {
		value, err := parseOlden(data)
		if err != nil {
			return err
		}
		dict, ok := value.(olden.BDict)
		if !ok {
			return fmt.Errorf("expected dictionary, got %T", value)
		}
		return decoder.UnmarshalBDict(dict, out)
	}},
	{"zeebo", func(data []byte, out *torrent) error { return zeebo.DecodeBytes(data, out) }},
	{"jackpal", func(data []byte, out *torrent) error { return jackpal.Unmarshal(bytes.NewReader(data), out) }},
}

func parseOlden(data []byte) (any, error) {
	p, err := olden.NewParser(data)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

// BenchmarkParse measures public APIs producing each library's generic tree.
// Input is already in memory; reader/parser creation happens on every iteration.
func BenchmarkParse(b *testing.B) {
	for _, f := range parseFixtures(b) {
		b.Run("case="+f.name, func(b *testing.B) {
			for _, impl := range parsers {
				b.Run("lib="+impl.name, func(b *testing.B) {
					// Validate even when regular tests are disabled with -run '^$'.
					checkParse(b, f, impl)
					b.ReportAllocs()
					b.SetBytes(int64(len(f.data)))
					for b.Loop() {
						_, err := impl.parse(f.data)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}

// BenchmarkUnmarshal measures the full bytes-to-struct path, including parsing
// for olden. Each operation starts with a fresh destination.
func BenchmarkUnmarshal(b *testing.B) {
	for _, f := range torrentFixtures(b) {
		b.Run("case="+f.name, func(b *testing.B) {
			for _, impl := range unmarshallers {
				b.Run("lib="+impl.name, func(b *testing.B) {
					checkUnmarshal(b, f, impl.unmarshal)
					b.ReportAllocs()
					b.SetBytes(int64(len(f.data)))
					for b.Loop() {
						var out torrent
						if err := impl.unmarshal(f.data, &out); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}

func TestParsersAgree(t *testing.T) {
	for _, f := range parseFixtures(t) {
		t.Run(f.name, func(t *testing.T) {
			t.Logf("input: %d bytes", len(f.data))
			for _, impl := range parsers {
				t.Run(impl.name, func(t *testing.T) { checkParse(t, f, impl) })
			}
		})
	}
}

func TestUnmarshallersAgree(t *testing.T) {
	for _, f := range torrentFixtures(t) {
		t.Run(f.name, func(t *testing.T) {
			for _, impl := range unmarshallers {
				t.Run(impl.name, func(t *testing.T) { checkUnmarshal(t, f, impl.unmarshal) })
			}
		})
	}
}

func checkParse(tb testing.TB, f fixture, impl parserImplementation) {
	tb.Helper()
	input := bytes.Clone(f.data)
	value, err := impl.parse(input)
	if err != nil {
		tb.Fatal(err)
	}
	if !reflect.DeepEqual(normalize(value), f.want) {
		tb.Fatal("parsed value differs from the expected fixture")
	}
	if !bytes.Equal(input, f.data) {
		tb.Fatal("parser modified its input")
	}
	if impl.name != "olden" {
		// Only olden's default parser borrows the caller's byte strings.
		clear(input)
		if !reflect.DeepEqual(normalize(value), f.want) {
			tb.Fatal("owned result changed after overwriting the input")
		}
	}
}

func checkUnmarshal(tb testing.TB, f torrentFixture, unmarshal func([]byte, *torrent) error) {
	tb.Helper()
	input := bytes.Clone(f.data)
	var out torrent
	if err := unmarshal(input, &out); err != nil {
		tb.Fatal(err)
	}
	if !reflect.DeepEqual(out, f.wantTorrent) {
		tb.Fatal("decoded struct differs from the expected fixture")
	}
	if !bytes.Equal(input, f.data) {
		tb.Fatal("unmarshaler modified its input")
	}
	clear(input)
	if !reflect.DeepEqual(out, f.wantTorrent) {
		tb.Fatal("decoded struct changed after overwriting the input")
	}
}

// normalize is used only for correctness checks, outside benchmark timing.
func normalize(value any) any {
	switch v := value.(type) {
	case olden.BInt:
		return int64(v)
	case olden.BString:
		return string(v)
	case olden.BList:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalize(item)
		}
		return out
	case olden.BDict:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalize(item)
		}
		return out
	default:
		return value
	}
}
