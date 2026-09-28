package main

// A minimal MessagePack encoder for the news formats. Maps keep their field order (omap), as the console's
// own records do; integral float64 values (numbers from JSON) are written as integers.

import (
	"encoding/binary"
	"math"
	"sort"
)

type kv struct {
	K string
	V any
}

// omap is an ordered map.
type omap []kv

func mpack(v any) []byte { return mpackAppend(nil, v) }

func mpackAppend(b []byte, v any) []byte {
	switch x := v.(type) {
	case nil:
		return append(b, 0xc0)
	case bool:
		if x {
			return append(b, 0xc3)
		}
		return append(b, 0xc2)
	case int:
		return mpackInt(b, int64(x))
	case int64:
		return mpackInt(b, x)
	case uint32:
		return mpackInt(b, int64(x))
	case uint64:
		if x > math.MaxInt64 {
			return binary.BigEndian.AppendUint64(append(b, 0xcf), x)
		}
		return mpackInt(b, int64(x))
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1<<53 {
			return mpackInt(b, int64(x))
		}
		return binary.BigEndian.AppendUint64(append(b, 0xcb), math.Float64bits(x))
	case string:
		n := len(x)
		switch {
		case n <= 31:
			b = append(b, 0xa0|byte(n))
		case n <= 0xff:
			b = append(b, 0xd9, byte(n))
		case n <= 0xffff:
			b = binary.BigEndian.AppendUint16(append(b, 0xda), uint16(n))
		default:
			b = binary.BigEndian.AppendUint32(append(b, 0xdb), uint32(n))
		}
		return append(b, x...)
	case []byte:
		n := len(x)
		switch {
		case n <= 0xff:
			b = append(b, 0xc4, byte(n))
		case n <= 0xffff:
			b = binary.BigEndian.AppendUint16(append(b, 0xc5), uint16(n))
		default:
			b = binary.BigEndian.AppendUint32(append(b, 0xc6), uint32(n))
		}
		return append(b, x...)
	case []string:
		b = mpackArrayHeader(b, len(x))
		for _, e := range x {
			b = mpackAppend(b, e)
		}
		return b
	case []any:
		b = mpackArrayHeader(b, len(x))
		for _, e := range x {
			b = mpackAppend(b, e)
		}
		return b
	case []omap:
		b = mpackArrayHeader(b, len(x))
		for _, e := range x {
			b = mpackAppend(b, e)
		}
		return b
	case omap:
		b = mpackMapHeader(b, len(x))
		for _, e := range x {
			b = mpackAppend(mpackAppend(b, e.K), e.V)
		}
		return b
	case map[string]any: // from JSON: sorted keys, for a stable encoding
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b = mpackMapHeader(b, len(x))
		for _, k := range keys {
			b = mpackAppend(mpackAppend(b, k), x[k])
		}
		return b
	}
	return append(b, 0xc0) // unsupported: nil
}

func mpackInt(b []byte, n int64) []byte {
	switch {
	case n >= 0 && n <= 0x7f:
		return append(b, byte(n))
	case n < 0 && n >= -32:
		return append(b, byte(n))
	case n >= 0 && n <= 0xff:
		return append(b, 0xcc, byte(n))
	case n >= 0 && n <= 0xffff:
		return binary.BigEndian.AppendUint16(append(b, 0xcd), uint16(n))
	case n >= 0 && n <= 0xffffffff:
		return binary.BigEndian.AppendUint32(append(b, 0xce), uint32(n))
	case n >= 0:
		return binary.BigEndian.AppendUint64(append(b, 0xcf), uint64(n))
	default:
		return binary.BigEndian.AppendUint64(append(b, 0xd3), uint64(n))
	}
}

func mpackArrayHeader(b []byte, n int) []byte {
	if n <= 15 {
		return append(b, 0x90|byte(n))
	}
	return binary.BigEndian.AppendUint16(append(b, 0xdc), uint16(n))
}

func mpackMapHeader(b []byte, n int) []byte {
	if n <= 15 {
		return append(b, 0x80|byte(n))
	}
	return binary.BigEndian.AppendUint16(append(b, 0xde), uint16(n))
}
