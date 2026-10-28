package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/olden/bencode"
	"github.com/olden/bencode/encoder"
)

func TestUnmarshalStrictTypes(t *testing.T) {
	type child struct {
		Size int `bencode:"size"`
	}
	type target struct {
		Count int8       `bencode:"count"`
		Total uint16     `bencode:"total"`
		Name  string     `bencode:"name"`
		Data  []byte     `bencode:"data"`
		Tags  []string   `bencode:"tags"`
		Tiers [][]string `bencode:"tiers"`
		Child child      `bencode:"child"`
	}
	for _, tt := range []struct {
		key   string
		value bencode.BValue
		path  string
	}{
		{"count", bencode.BString("7"), "count"},
		{"count", bencode.BInt(128), "count"},
		{"count", bencode.BInt(-129), "count"},
		{"total", bencode.BInt(-1), "total"},
		{"total", bencode.BInt(65536), "total"},
		{"name", bencode.BInt(7), "name"},
		{"data", bencode.BList{bencode.BInt(7)}, "data"},
		{"tags", bencode.BList{bencode.BString("ok"), bencode.BInt(7)}, "[1]"},
		{"tiers", bencode.BList{bencode.BList{bencode.BInt(7)}}, "[0]"},
		{"child", bencode.BString("bad"), "child"},
		{"child", bencode.BDict{"size": bencode.BString("bad")}, "size"},
		{"name", nil, "name"},
	} {
		t.Run(tt.key, func(t *testing.T) {
			var out target
			err := UnmarshalBDict(bencode.BDict{tt.key: tt.value}, &out)
			if err == nil || !strings.Contains(err.Error(), tt.path) {
				t.Fatalf("got %v, want error containing %q", err, tt.path)
			}
		})
	}
}

func TestUnmarshalTypedCollectionsAndTags(t *testing.T) {
	type count int16
	type octet uint8
	type label string
	type entry struct {
		ID count `bencode:"id"`
	}
	type target struct {
		Name   label           `bencode:"name"`
		Bytes  []octet         `bencode:"bytes"`
		Hash   [2]byte         `bencode:"hash"`
		Items  []entry         `bencode:"items"`
		Counts map[label]count `bencode:"counts"`
		Node   *entry          `bencode:"node"`
		Raw    bencode.BDict   `bencode:"raw"`
		Ignore int             `bencode:"-"`
	}
	input := bencode.BDict{
		"name":    bencode.BString("cat"),
		"bytes":   bencode.BString{0, 255},
		"hash":    bencode.BString{1, 2},
		"items":   bencode.BList{bencode.BDict{"id": bencode.BInt(7)}},
		"counts":  bencode.BDict{"x": bencode.BInt(-2)},
		"node":    bencode.BDict{"id": bencode.BInt(8)},
		"raw":     bencode.BDict{"list": bencode.BList{bencode.BInt(3)}},
		"-":       bencode.BString("ignored"),
		"unknown": bencode.BInt(1),
	}
	var out target
	if err := UnmarshalBDict(input, &out); err != nil {
		t.Fatal(err)
	}
	want := target{
		Name: "cat", Bytes: []octet{0, 255}, Hash: [2]byte{1, 2},
		Items: []entry{{ID: 7}}, Counts: map[label]count{"x": -2},
		Node: &entry{ID: 8}, Raw: bencode.BDict{"list": bencode.BList{bencode.BInt(3)}},
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("got %#v, want %#v", out, want)
	}
	encoded, err := encoder.Encode(out)
	if err != nil {
		t.Fatal(err)
	}
	p, err := bencode.NewParser(encoded)
	if err != nil {
		t.Fatal(err)
	}
	value, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	var again target
	if err := UnmarshalBDict(value.(bencode.BDict), &again); err != nil || !reflect.DeepEqual(out, again) {
		t.Fatalf("struct round trip failed: %#v, %v", again, err)
	}
}

func TestUnmarshalLeavesOutputUnchangedOnError(t *testing.T) {
	type child struct {
		Value int `bencode:"value"`
	}
	out := struct {
		Child *child   `bencode:"child"`
		List  []string `bencode:"list"`
		Last  int      `bencode:"last"`
	}{Child: &child{Value: 1}, List: []string{"old"}, Last: 2}
	oldChild, oldList := out.Child, out.List
	err := UnmarshalBDict(bencode.BDict{
		"child": bencode.BDict{"value": bencode.BInt(9)},
		"list":  bencode.BList{bencode.BString("new")},
		"last":  bencode.BString("wrong"),
	}, &out)
	if err == nil {
		t.Fatal("accepted wrong field type")
	}
	if out.Child != oldChild || out.Child.Value != 1 || out.List[0] != "old" || oldList[0] != "old" || out.Last != 2 {
		t.Fatalf("partially modified output: %#v", out)
	}
}

func TestUnmarshalInvalidDestinations(t *testing.T) {
	for _, out := range []any{nil, (*struct{})(nil), struct{}{}, new(int)} {
		if err := UnmarshalBDict(bencode.BDict{}, out); err == nil {
			t.Fatalf("accepted destination %T", out)
		}
	}
	var unsupported struct {
		Value bool `bencode:"value"`
	}
	if err := UnmarshalBDict(bencode.BDict{"value": bencode.BInt(1)}, &unsupported); err == nil {
		t.Fatal("silently ignored unsupported field")
	}
	var duplicates struct {
		A int `bencode:"key"`
		B int `bencode:"key"`
	}
	if err := UnmarshalBDict(bencode.BDict{"key": bencode.BInt(1)}, &duplicates); err == nil {
		t.Fatal("accepted duplicate field tags")
	}
	cyclic := bencode.BDict{}
	cyclic["self"] = cyclic
	var recursive struct {
		Value any `bencode:"value"`
	}
	if err := UnmarshalBDict(bencode.BDict{"value": cyclic}, &recursive); err == nil {
		t.Fatal("accepted cyclic input")
	}
	var pointer any
	pointer = &pointer
	if err := UnmarshalBDict(bencode.BDict{"value": pointer}, &recursive); err == nil {
		t.Fatal("accepted unsupported pointer source")
	}
	var options struct {
		A int `bencode:"a,unknown"`
	}
	if err := UnmarshalBDict(bencode.BDict{"a": bencode.BInt(1)}, &options); err == nil {
		t.Fatal("silently accepted unsupported tag options")
	}
}

func TestUnmarshalDepthLimit(t *testing.T) {
	for _, depth := range []int{128, 129} {
		var value bencode.BValue = bencode.BInt(1)
		// The outer dictionary itself counts as one container.
		for level := 1; level < depth; level++ {
			value = bencode.BList{value}
		}
		var out struct {
			Value any `bencode:"value"`
		}
		err := UnmarshalBDict(bencode.BDict{"value": value}, &out)
		if (err != nil) != (depth > 128) {
			t.Fatalf("depth %d: error = %v", depth, err)
		}
	}
}

func TestUnmarshalCopiesBytesAndPreservesAbsentFields(t *testing.T) {
	input := bencode.BString{1, 2}
	out := struct {
		Data []byte `bencode:"data"`
		Name string `bencode:"name"`
	}{Name: "keep"}
	if err := UnmarshalBDict(bencode.BDict{"data": input}, &out); err != nil {
		t.Fatal(err)
	}
	input[0] = 9
	if out.Data[0] != 1 || out.Name != "keep" {
		t.Fatalf("aliased input or lost absent field: %#v", out)
	}
}
