package encoder

import (
	"strings"
	"testing"

	"github.com/olden/bencode"
)

func TestEncodeGoTypes(t *testing.T) {
	type count int16
	type octet uint8
	type label string
	for _, tt := range []struct {
		name string
		in   any
		want string
	}{
		{"named_integer", count(-7), "i-7e"},
		{"unsigned_integer", uint32(42), "i42e"},
		{"named_bytes", []octet{0, 255}, "2:\x00\xff"},
		{"byte_array", [3]byte{'c', 'a', 't'}, "3:cat"},
		{"typed_list", []int16{1, -2}, "li1ei-2ee"},
		{"named_map_keys", map[label]count{"z": 2, "a": 1}, "d1:ai1e1:zi2ee"},
		{"nil_bytes", []byte(nil), "0:"},
		{"nil_list", bencode.BList(nil), "le"},
		{"nil_map", bencode.BDict(nil), "de"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.in)
			if err != nil || string(got) != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestEncodeStructTags(t *testing.T) {
	input := struct {
		Name   string `bencode:"name"`
		Ignore func() `bencode:"-"`
		hidden bool
	}{Name: "cat", Ignore: func() {}, hidden: true}
	got, err := Encode(&input)
	if err != nil || string(got) != "d4:name3:cate" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestEncodeErrors(t *testing.T) {
	cyclicMap := bencode.BDict{}
	cyclicMap["self"] = cyclicMap
	cyclicList := make(bencode.BList, 1)
	cyclicList[0] = cyclicList
	var cyclicPointer any
	cyclicPointer = &cyclicPointer
	for _, tt := range []struct {
		name string
		in   any
	}{
		{"nil", nil},
		{"nil_pointer", (*int)(nil)},
		{"nested_nil", []any{1, nil}},
		{"float", 1.5},
		{"bool", true},
		{"numeric_map_key", map[int]string{1: "x"}},
		{"unsigned_overflow", ^uint64(0)},
		{"duplicate_tags", struct {
			A int `bencode:"key"`
			B int `bencode:"key"`
		}{}},
		{"unsupported_tag_option", struct {
			A int `bencode:"a,unknown"`
		}{}},
		{"cyclic_map", cyclicMap},
		{"cyclic_list", cyclicList},
		{"cyclic_pointer", cyclicPointer},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.in)
			if err == nil || got != nil {
				t.Fatalf("expected error and no partial output, got %q, %v", got, err)
			}
		})
	}
}

func TestEncodeDepthLimit(t *testing.T) {
	for _, depth := range []int{128, 129} {
		var value any = bencode.BInt(0)
		for i := 0; i < depth; i++ {
			value = bencode.BList{value}
		}
		got, err := Encode(value)
		if depth == 129 {
			if err == nil {
				t.Fatal("accepted excessive nesting")
			}
			continue
		}
		want := strings.Repeat("l", depth) + "i0e" + strings.Repeat("e", depth)
		if err != nil || string(got) != want {
			t.Fatalf("valid depth failed: %v", err)
		}
	}
}
