// Package bencode parses one complete bencoded document into BInt, BString,
// BList and BDict values. It rejects malformed numbers, integer overflow,
// duplicate or unsorted dictionary keys, trailing data and nesting deeper
// than MaxDepth. Integers are limited to int64; strings contain arbitrary bytes.
//
// Create a Parser with NewParser and call Parse once. Parsed byte strings and
// optional Parser.RawInfo bytes reference the input buffer, which must remain
// unchanged while those values are in use. Parsing does not validate torrent
// metadata fields or impose a total input-size limit; callers should bound
// untrusted input before loading it into memory.
//
// The encoder subpackage serializes BValue trees and supported Go values.
// The decoder subpackage maps BDict values to structs with strict type checks.
package bencode
