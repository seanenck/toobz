package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := unpack(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func toUint8(s string) []uint8 {
	var r []uint8
	for _, chr := range s {
		r = append(r, uint8(chr))
	}

	return r
}

func unpack() error {
	in := flag.String("in", "", "input file")
	out := flag.String("out", "", "output file")
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

	type check struct {
		name  string
		left  []uint8
		right []uint8
	}
	for _, c := range []check{
		{"msdos magic", hdr.MSDosMagic[:], toUint8("MZ")},
		// aka \xcd\x23\x82\x81
		{"linux magic", hdr.LinuxMagic[:], []uint8{205, 35, 130, 129}},
		{"zimg", hdr.ZImg[:], toUint8("zimg")},
	} {
		l := fmt.Sprintf("%v", c.left)
		r := fmt.Sprintf("%v", c.right)
		if l != r {
			return fmt.Errorf("%s is invalid, %s != %s", c.name, l, r)
		}
	}
	if hdr.PayloadOffset == 0 || hdr.PayloadSize == 0 {
		return errors.New("payload size/offset is zero")
	}
	size := len(b)
	if int(hdr.PayloadOffset+hdr.PayloadSize) > size {
		return errors.New("invalid offset/payload, beyond size")
	}
	sub := b[hdr.PayloadOffset:]
	return os.WriteFile(output, sub, 0o644)
}
