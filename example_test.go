package bencode_test

import (
	"crypto/sha1"
	"fmt"

	"github.com/olden/bencode"
	"github.com/olden/bencode/decoder"
	"github.com/olden/bencode/encoder"
)

func Example() {
	p, err := bencode.NewParser([]byte("d4:name3:cat4:sizei7ee"))
	if err != nil {
		panic(err)
	}
	value, err := p.Parse()
	if err != nil {
		panic(err)
	}
	dict, ok := value.(bencode.BDict)
	if !ok {
		panic("expected dictionary")
	}
	var out struct {
		Name string `bencode:"name"`
		Size int    `bencode:"size"`
	}
	if err := decoder.UnmarshalBDict(dict, &out); err != nil {
		panic(err)
	}
	encoded, err := encoder.Encode(out)
	if err != nil {
		panic(err)
	}
	fmt.Println(out.Name, out.Size)
	fmt.Println(string(encoded))
	// Output:
	// cat 7
	// d4:name3:cat4:sizei7ee
}

func ExampleParser() {
	// Only demonstrates raw info extraction, not a complete torrent file.
	data := []byte("d4:infod4:name3:catee")
	p, err := bencode.NewParser(data)
	if err != nil {
		panic(err)
	}
	var rawInfo []byte
	p.RawInfo = &rawInfo
	value, err := p.Parse()
	if err != nil {
		panic(err)
	}
	root, ok := value.(bencode.BDict)
	if !ok {
		panic("expected root dictionary")
	}
	if _, ok := root["info"].(bencode.BDict); !ok {
		panic("expected info dictionary")
	}
	fmt.Println(string(rawInfo))
	fmt.Println(len(sha1.Sum(rawInfo)))
	// Output:
	// d4:name3:cate
	// 20
}
