package encoder

import "testing"

func TestEncodeInterface(t *testing.T) {
	input := map[string]interface{}{
		"title":  "example",
		"length": 42,
		"tags":   []interface{}{"go", "bencode"},
		"meta": map[string]interface{}{
			"author": "alice",
		},
	}

	expected := "d6:lengthi42e4:metad6:author5:alicee4:tagsl2:go7:bencodee5:title7:examplee"

	output, err := Encode(input)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if string(output) != expected {
		t.Errorf("unexpected encoding:\n got:  %q\n want: %q", string(output), expected)
	}
}

type testStruct struct {
	Title  string       `bencode:"title"`
	Length int          `bencode:"length"`
	Tags   []string     `bencode:"tags"`
	Meta   nestedStruct `bencode:"meta"`
	Inline struct {
		Title string `bencode:"tit"`
	} `bencode:"inline"`
}

type nestedStruct struct {
	Author string `bencode:"author"`
}

func TestEncodeStruct(t *testing.T) {
	input := testStruct{
		Title:  "example",
		Length: 42,
		Tags:   []string{"go", "bencode"},
		Meta: nestedStruct{
			Author: "alice",
		},
		Inline: struct {
			Title string `bencode:"tit"`
		}{
			Title: "inline-title",
		},
	}

	expected := "d6:inlined3:tit12:inline-titlee6:lengthi42e4:metad6:author5:alicee4:tagsl2:go7:bencodee5:title7:examplee"

	output, err := Encode(input)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if string(output) != expected {
		t.Errorf("unexpected encoding:\n got:  %q\n want: %q", string(output), expected)
	}
}

func BenchmarkEncodeStruct(b *testing.B) {
	input := testStruct{
		Title:  "example",
		Length: 42,
		Tags:   []string{"go", "bencode"},
		Meta: nestedStruct{
			Author: "alice",
		},
		Inline: struct {
			Title string `bencode:"tit"`
		}{
			Title: "inline-title",
		},
	}

	for i := 0; i < b.N; i++ {
		_, _ = Encode(input)
	}
}
