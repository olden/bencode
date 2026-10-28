package decoder

import (
	"testing"

	"torantulla/pkg/bencode"
)

type testStruct struct {
	Title  string       `bencode:"title"`
	Length int          `bencode:"length"`
	Tags   []string     `bencode:"tags"`
	Meta   nestedStruct `bencode:"meta"`
	Inline struct {
		Title string `bencode:"title"`
	} `bencode:"inline"`
}

type nestedStruct struct {
	Author string `bencode:"author"`
}

func TestUnmarshalBDict(t *testing.T) {
	input := bencode.BDict{
		"title":  bencode.BString([]byte("example")),
		"length": bencode.BInt(42),
		"tags": bencode.BList{
			bencode.BString([]byte("go")),
			bencode.BString([]byte("bencode")),
		},
		"meta": bencode.BDict{
			"author": bencode.BString([]byte("alice")),
		},
		"inline": bencode.BDict{
			"title": bencode.BString([]byte("test")),
		},
	}

	var out testStruct
	err := UnmarshalBDict(input, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Title != "example" {
		t.Errorf("Title mismatch: got %q", out.Title)
	}
	if out.Length != 42 {
		t.Errorf("Length mismatch: got %d", out.Length)
	}
	if len(out.Tags) != 2 || out.Tags[0] != "go" || out.Tags[1] != "bencode" {
		t.Errorf("Tags mismatch: got %#v", out.Tags)
	}
	if out.Meta.Author != "alice" {
		t.Errorf("Meta.Author mismatch: got %q", out.Meta.Author)
	}
	if out.Inline.Title != "test" {
		t.Errorf("Inline.Title mismatch: got %q", out.Meta.Author)
	}
}

func BenchmarkUnmarshalBDict(b *testing.B) {
	input := bencode.BDict{
		"title":  bencode.BString([]byte("example")),
		"length": bencode.BInt(42),
		"tags": bencode.BList{
			bencode.BString([]byte("go")),
			bencode.BString([]byte("bencode")),
		},
		"meta": bencode.BDict{
			"author": bencode.BString([]byte("alice")),
		},
	}

	for i := 0; i < b.N; i++ {
		var out testStruct
		if err := UnmarshalBDict(input, &out); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
