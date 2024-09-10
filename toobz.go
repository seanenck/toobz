// Package main unpacks an EFI zboot file
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var errInvalidCheck = errors.New("invalid data")

type check struct {
	name  string
	left  []uint8
	right []uint8
}

func (c check) verify() error {
	l := fmt.Sprintf("%v", c.left)
	r := fmt.Sprintf("%v", c.right)
	if l != r {
		return errors.Join(errInvalidCheck, fmt.Errorf("%s is invalid, %s != %s", c.name, l, r))
	}
	return nil
}

func main() {
	if err := unpack(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func toUint8(s string) []uint8 {
	return toUint8Padded(s, 0)
}

func toUint8Padded(s string, to int) []uint8 {
	var r []uint8
	for _, chr := range s {
		r = append(r, uint8(chr))
	}

	for len(r) < to {
		r = append(r, 0)
	}

	return r
}

func decompressGunzip(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func unpack() error {
	in := flag.String("in", "", "input file")
	out := flag.String("out", "", "output file")
	decompress := flag.Bool("decompress", false, "decompress the resulting data")
	flag.Parse()
	input := *in
	output := *out
	if input == "" || output == "" {
		return errors.New("input/output files required")
	}

	b, err := os.ReadFile(input)
	if err != nil {
		return err
	}

	type header struct {
		MSDosMagic      [2]uint8
		Reserved0       [2]uint8
		ZImg            [4]uint8
		PayloadOffset   uint32
		PayloadSize     uint32
		Reserved1       [8]uint8
		CompressionType [32]uint8
		LinuxMagic      [4]uint8
		PEHeaderOffset  uint32
	}

	hdr := header{}
	n, err := binary.Decode(b, binary.LittleEndian, &hdr)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("unable to decode, no data?")
	}

	for _, c := range []check{
		{"msdos magic", hdr.MSDosMagic[:], toUint8("MZ")},
		// aka \xcd\x23\x82\x81
		{"linux magic", hdr.LinuxMagic[:], []uint8{205, 35, 130, 129}},
		{"zimg", hdr.ZImg[:], toUint8("zimg")},
	} {
		if err := c.verify(); err != nil {
			return err
		}
	}
	if hdr.PayloadOffset == 0 || hdr.PayloadSize == 0 {
		return errors.New("payload size/offset is zero")
	}
	size := len(b)
	if int(hdr.PayloadOffset+hdr.PayloadSize) > size {
		return errors.New("invalid offset/payload, beyond size")
	}
	const arm64MagicOffset = 56
	sub := b[hdr.PayloadOffset : hdr.PayloadOffset+hdr.PayloadSize]
	if *decompress {
		found := false
		t := fmt.Sprintf("%v", hdr.CompressionType[:])
		for k, v := range map[string](func([]byte) ([]byte, error)){
			"gzip": decompressGunzip,
		} {
			if t == fmt.Sprintf("%v", toUint8Padded(k, 32)) {
				found = true
				d, err := v(sub)
				if err != nil {
					return err
				}
				sub = d
			}
		}
		if !found {
			return fmt.Errorf("unknown compression type: %s", t)
		}

		subSize := len(sub)
		if subSize < arm64MagicOffset {
			return fmt.Errorf("invalid response payload: %d", subSize)
		}
		val := sub[arm64MagicOffset : arm64MagicOffset+4]
		knownType := false
		for k, v := range map[string][]uint8{
			"arm":  append(toUint8("ARM"), 100),
			"risc": append(toUint8("RSC"), 5),
		} {
			err := (check{k, val, v}).verify()
			if err == nil {
				knownType = true
				break
			}
			if !errors.Is(err, errInvalidCheck) {
				return err
			}
		}
		if !knownType {
			return fmt.Errorf("unknown payload type: %v", val)
		}
	}
	return os.WriteFile(output, sub, 0o644)
}
