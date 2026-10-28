package benchmarks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/olden/bencode/encoder"
)

type fixture struct {
	name string
	data []byte
	want any
}

type torrent struct {
	Announce string      `bencode:"announce"`
	Comment  string      `bencode:"comment"`
	Info     torrentInfo `bencode:"info"`
}

type torrentInfo struct {
	Files       []torrentFile `bencode:"files"`
	Length      int64         `bencode:"length"`
	Name        string        `bencode:"name"`
	PieceLength int64         `bencode:"piece length"`
	Pieces      string        `bencode:"pieces"`
}

type torrentFile struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
}

type torrentFixture struct {
	fixture
	wantTorrent torrent
}

func encodedFixture(tb testing.TB, name string, value any) fixture {
	tb.Helper()
	data, err := encoder.Encode(value)
	if err != nil {
		tb.Fatal(err)
	}
	return fixture{name: name, data: data, want: value}
}

func parseFixtures(tb testing.TB) []fixture {
	tb.Helper()
	integers := make([]any, 256)
	for i := range integers {
		integers[i] = int64(i - 128)
	}
	var nested any = "leaf"
	for range 64 {
		nested = []any{nested}
	}
	fixtures := []fixture{
		encodedFixture(tb, "SmallDict", map[string]any{"name": "cat", "size": int64(7)}),
		encodedFixture(tb, "IntegerList256", integers),
		encodedFixture(tb, "Binary64KiB", strings.Repeat("\x00\xff\x80bencode", 6554)[:65536]),
		encodedFixture(tb, "NestedList64", nested),
	}
	for _, f := range torrentFixtures(tb) {
		fixtures = append(fixtures, f.fixture)
	}
	return fixtures
}

func torrentFixtures(tb testing.TB) []torrentFixture {
	tb.Helper()
	// Synthetic metadata: no files or network access are needed. These bytes
	// resemble piece hashes but are not hashes of real payloads.
	const pieceLength int64 = 256 * 1024
	const pieceCount = 1024
	pieces := make([]byte, 20*pieceCount)
	for i := range pieces {
		pieces[i] = byte(i*37 + 11)
	}
	var fixtures []torrentFixture
	for _, multi := range []bool{false, true} {
		name := "SingleFile"
		want := torrent{
			Announce: "https://tracker.example.invalid/announce",
			Comment:  "Synthetic benchmark fixture",
			Info: torrentInfo{
				Name:        "single.bin",
				Length:      pieceLength * pieceCount,
				PieceLength: pieceLength,
				Pieces:      string(pieces),
			},
		}
		info := map[string]any{
			"length":       want.Info.Length,
			"name":         want.Info.Name,
			"piece length": want.Info.PieceLength,
			"pieces":       want.Info.Pieces,
		}
		if multi {
			name = "MultiFile64"
			want.Info.Name = "collection"
			want.Info.Length = 0
			delete(info, "length")
			info["name"] = want.Info.Name
			files := make([]any, 64)
			for i := range files {
				file := torrentFile{
					Length: pieceLength * pieceCount / int64(len(files)),
					Path:   []string{fmt.Sprintf("dir%02d", i/8), fmt.Sprintf("file%03d.bin", i)},
				}
				want.Info.Files = append(want.Info.Files, file)
				files[i] = map[string]any{
					"length": file.Length,
					"path":   []any{file.Path[0], file.Path[1]},
				}
			}
			info["files"] = files
		}
		value := map[string]any{
			"announce": want.Announce,
			"comment":  want.Comment,
			"info":     info,
		}
		fixtures = append(fixtures, torrentFixture{
			fixture:     encodedFixture(tb, name, value),
			wantTorrent: want,
		})
	}
	return fixtures
}
