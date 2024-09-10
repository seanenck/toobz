// Package toobz implements the means to unpack an EFI zboot file
package toobz

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// ARM64MagicOffset the decompress offset to find magic value to detect ARM/RISC types
	ARM64MagicOffset = 56
)

var (
	// ErrIsInvalidContent indicates a generic error for bad header information
	ErrIsInvalidContent = errors.New("invalid content data")
	// LinuxMagic is the magic number for the Linux header field
	LinuxMagic = Datum{payload: "Linux", raw: []uint8{205, 35, 130, 129}}
	// ARM is the magic value to indicate ARM
	ARM = Datum{payload: "ARM", addByte: 100}
	// RISC is the magic value to indicate RISC
	RISC = Datum{payload: "RSC", addByte: 5}
	// Gzip indicates a gzip payload to decompress
	Gzip = Datum{payload: "gzip", padding: 32}
	// MSDOSMagic is the magic indicator for the MSDOS field
	MSDOSMagic = Datum{payload: "MZ"}
	// ZImg is the indicator that the header is for zimg
	ZImg = Datum{payload: "zimg"}
)

type (
	// Datum are special fields within the file to assist in reading/loading information
	Datum struct {
		payload string
		padding int
		addByte uint8
		raw     []uint8
	}
	check struct {
		left  []uint8
		right Datum
	}
	// Unpacker handles the core reading/parsing of the header/body
	Unpacker struct {
		Decompress bool
		ParseBody  bool
	}
	// Header is the parsed zboot header information
	Header struct {
		MSDOSMagic      [2]uint8
		Reserved0       [2]uint8
		ZImg            [4]uint8
		PayloadOffset   uint32
		PayloadSize     uint32
		Reserved1       [8]uint8
		CompressionType [32]uint8
		LinuxMagic      [4]uint8
		PEHeaderOffset  uint32
	}
	// BootInfo is the wrapper around the parsed header and body segment (if requested)
	BootInfo struct {
		Header   Header
		body     []byte
		unpacker Unpacker
	}
)

// Value will get the raw string name value for the data item
func (d Datum) Value() string {
	return d.payload
}

// Data will get the expected data value for an item (the actual value of interest)
func (d Datum) Data() []uint8 {
	if len(d.raw) > 0 {
		return d.raw
	}
	b := toUint8Padded(d.payload, d.padding)
	if d.addByte > 0 {
		b = append(b, d.addByte)
	}
	return b
}

func (c check) verify() error {
	l := fmt.Sprintf("%v", c.left)
	r := fmt.Sprintf("%v", c.right.Data())
	if l != r {
		return errors.Join(ErrIsInvalidContent, fmt.Errorf("%s invalid data: %s != %s", c.right.Value(), l, r))
	}
	return nil
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

// ReadInfo will read boot information from an input reader
func (u Unpacker) ReadInfo(r *bytes.Reader) (BootInfo, error) {
	if r == nil {
		return BootInfo{}, errors.New("reader is nil")
	}

	hdr := Header{}
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return BootInfo{}, err
	}

	for _, c := range []check{
		{hdr.MSDOSMagic[:], MSDOSMagic},
		{hdr.LinuxMagic[:], LinuxMagic},
		{hdr.ZImg[:], ZImg},
	} {
		if err := c.verify(); err != nil {
			return BootInfo{}, err
		}
	}
	if hdr.PayloadOffset == 0 || hdr.PayloadSize == 0 {
		return BootInfo{}, errors.New("payload size/offset is zero")
	}
	size := r.Len()
	if int(hdr.PayloadOffset+hdr.PayloadSize) > size {
		return BootInfo{}, errors.New("invalid offset/payload, beyond size")
	}
	var sub []byte
	if u.ParseBody {
		sub = make([]byte, hdr.PayloadSize)
		n, err := r.ReadAt(sub, int64(hdr.PayloadOffset))
		if err != nil {
			return BootInfo{}, err
		}
		if n == 0 {
			return BootInfo{}, errors.New("invalid seek, zero")
		}
	}
	return BootInfo{Header: hdr, body: sub, unpacker: u}, nil
}

// Body will get the parsed body
func (info BootInfo) Body() []byte {
	return info.body
}

// Write will write (and optionally decompress prior) the payload of the file
func (info BootInfo) Write(w io.Writer) error {
	if len(info.body) == 0 {
		return errors.New("no body")
	}
	sub := info.body
	if info.unpacker.Decompress {
		found := false
		t := fmt.Sprintf("%v", info.Header.CompressionType[:])
		type decompressor struct {
			bodyType []uint8
			fxn      func([]byte) ([]byte, error)
		}
		for _, v := range []decompressor{
			{Gzip.Data(), decompressGunzip},
		} {
			if t == fmt.Sprintf("%v", v.bodyType) {
				found = true
				d, err := v.fxn(sub)
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
		if subSize < ARM64MagicOffset {
			return fmt.Errorf("invalid response payload: %d", subSize)
		}
		val := sub[ARM64MagicOffset : ARM64MagicOffset+4]
		knownType := false
		for _, c := range []Datum{ARM, RISC} {
			err := (check{val, c}).verify()
			if err == nil {
				knownType = true
				break
			}
			if !errors.Is(err, ErrIsInvalidContent) {
				return err
			}
		}
		if !knownType {
			return fmt.Errorf("unknown payload type: %v", val)
		}
	}
	_, err := w.Write(sub)
	return err
}
