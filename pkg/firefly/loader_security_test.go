/*
Security regression fixtures adapted from klauspost/compress.
Copyright (c) 2012 The Go Authors. All rights reserved.
Copyright (c) 2019 Klaus Post. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

  - Redistributions of source code must retain the above copyright

notice, this list of conditions and the following disclaimer.
  - Redistributions in binary form must reproduce the above

copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
  - Neither the name of Google Inc. nor the names of its

contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package firefly

import (
	"bytes"
	"cyrene-launcher/pkg/firefly/pb"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func compressedFixture(t *testing.T, data []byte) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	return enc.EncodeAll(data, nil)
}

func writeCompressedFixture(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.zst")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompressedProtoCompatibility(t *testing.T) {
	chunk := &pb.ChunkProto{Assets: []*pb.AssetChunkProperty{{AssetName: "fixture.bin", AssetSize: 123}}}
	manifest := &pb.ManifestProto{Assets: []*pb.AssetManifestProperty{{AssetName: "fixture.bin", AssetSize: 123}}}
	for _, tc := range []struct {
		name string
		want proto.Message
		load func(string) (proto.Message, error)
	}{
		{"chunk", chunk, func(p string) (proto.Message, error) { return LoadChunkProto(p) }},
		{"manifest", manifest, func(p string) (proto.Message, error) { return LoadManifestProto(p) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := proto.Marshal(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			encoded := compressedFixture(t, data)
			got, err := tc.load(writeCompressedFixture(t, encoded))
			if err != nil || !proto.Equal(got, tc.want) {
				t.Fatalf("round trip: got %v, error %v", got, err)
			}
			for name, bad := range map[string][]byte{
				"invalid-zstd":     []byte("not a zstd frame"),
				"truncated":        encoded[:len(encoded)-1],
				"invalid-protobuf": compressedFixture(t, []byte{0xff}),
			} {
				t.Run(name, func(t *testing.T) {
					if _, err := tc.load(writeCompressedFixture(t, bad)); err == nil {
						t.Fatal("expected decode error")
					}
				})
			}
			if _, err := tc.load(filepath.Join(t.TempDir(), "missing.zst")); err == nil {
				t.Fatal("expected file error")
			}
		})
	}
}

// GO-2026-6865: bounded regression adapted from the upstream BSD-3-Clause
// TestDecoderHuffWeightsTableLog (commit 7f5d8528720c895a6106f201719f87eb030a565d).
// Run only with the patched dependency: the old streaming decoder may panic
// from a worker goroutine. No real game assets are used.
func TestZstdRejectsHuffmanWeightOverflow(t *testing.T) {
	oneFrame := slices.Concat(
		[]byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x38, 0x74, 0x06, 0x00, 0x42, 0xb8, 0x32, 0x0d, 0x1a, 0x00, 0x00, 0x00, 0x90, 0xaf, 0xfa, 0xff, 0x03, 0x00, 0x80, 0xff, 0x7f},
		bytes.Repeat([]byte{0xa5, 0x94, 0x52, 0x4a, 0x29}, 37),
		[]byte{0xa5, 0x94, 0x12, 0x00, 0xad, 0x56, 0x03, 0xee, 0xff, 0xdf, 0xb3, 0x1a, 0x1c, 0x1a, 0x00, 0xe0, 0xff, 0x1f, 0x0f, 0xb1, 0x9c, 0x5a, 0xcf, 0x4d, 0xe4, 0x1d, 0x7b, 0xab, 0x99, 0x5e, 0x01, 0x51, 0xa4, 0x73, 0xd8, 0xb1, 0xf3, 0x39, 0x4b, 0x67, 0xf6, 0xab, 0x1a, 0xab, 0x1a, 0xab, 0x1a},
		bytes.Repeat([]byte{0x8c, 0x31, 0xc6, 0x18, 0x63}, 1365),
		[]byte{0x8c, 0x31},
		bytes.Repeat([]byte{0x63, 0x8c, 0x31, 0xc6, 0x18}, 1365),
		[]byte{0x63, 0x2c},
		bytes.Repeat([]byte{0x31, 0xc6, 0x18, 0x63, 0x8c}, 1365),
		[]byte{0x31, 0x46},
		bytes.Repeat([]byte{0x31, 0xc6, 0x18, 0x63, 0x8c}, 1365),
		[]byte{0x31, 0x06, 0x00},
	)
	twoFrames := slices.Concat(oneFrame[:6], []byte{0x75, 0x06, 0x00}, oneFrame[9:215], oneFrame[:6], oneFrame[215:])
	for _, input := range [][]byte{oneFrame, twoFrames} {
		dec, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(8<<20))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dec.DecodeAll(input, nil); err == nil {
			t.Error("DecodeAll accepted invalid weights")
		}
		if err := dec.Reset(bytes.NewReader(input)); err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, io.LimitReader(dec, 1<<20)); err == nil {
			t.Error("stream accepted invalid weights")
		}
		dec.Close()
		// Exercise both application loaders with the same small invalid input.
		path := writeCompressedFixture(t, input)
		if _, err := LoadChunkProto(path); err == nil {
			t.Error("chunk loader accepted invalid weights")
		}
		if _, err := LoadManifestProto(path); err == nil {
			t.Error("manifest loader accepted invalid weights")
		}
	}
}

// GO-2026-5841: the malicious repeat is a varint, not an allocated huge buffer.
// Derived from upstream TestNewDictRepeatOverflow (BSD-3-Clause), commit 8668e357.
func TestS2RejectsDictionaryRepeatOverflow(t *testing.T) {
	for _, repeat := range []uint64{1 << 31, math.MaxUint32, math.MaxInt64 + 1, math.MaxUint64} {
		var header [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(header[:], repeat)
		data := append(append([]byte{}, header[:n]...), make([]byte, s2.MinDictSize)...)
		if s2.NewDict(data) != nil {
			t.Fatalf("accepted out-of-range repeat %d", repeat)
		}
	}
	var header [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(header[:], uint64(s2.MinDictSize))
	if s2.NewDict(append(append([]byte{}, header[:n]...), make([]byte, s2.MinDictSize)...)) == nil {
		t.Fatal("valid boundary dictionary was rejected")
	}
}
