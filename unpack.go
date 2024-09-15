// Package toobz implements the means to unpack an EFI zboot file
package toobz

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
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

const (
	// ParseBodyOption will enable parsing the body of the input
	ParseBodyOption ReadInfoOption = iota
	// DebugReadInfoOption enable debug mode for reading
	DebugReadInfoOption
)

const (
	// DecompressOption handles writing the decompressed output (not the compressed output) after extraction
	DecompressOption UnpackOption = iota
	// DebugUnpackOption enable debug mode for unpacking
	DebugUnpackOption
)

type (
	// ReadInfoOption defines reading options
	ReadInfoOption int
	// UnpackOption handles writing options
	UnpackOption int

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
		header Header
		body   []byte
	}
	// Reader is the interface to read underlying boot information from an input
	Reader interface {
		io.ReaderAt
		Len() int
		io.Reader
	}
	// Package is data ready to unpack/be used
	Package interface {
		Body() []byte
		Headers() Header
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
	if slices.Compare(c.left, c.right.Data()) != 0 {
		l := fmt.Sprintf("%v", c.left)
		r := fmt.Sprintf("%v", c.right.Data())
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

func debugStatement(msg string) {
	fmt.Fprintf(os.Stderr, "[debug] %s\n", msg)
}

// ReadInfo will read boot information from an input reader
func ReadInfo(r Reader, opts ...ReadInfoOption) (BootInfo, error) {
	if r == nil {
		return BootInfo{}, errors.New("reader is nil")
	}

	hdr := Header{}
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return BootInfo{}, err
	}

	debug := slices.Contains(opts, DebugReadInfoOption)
	if debug {
		debugStatement(fmt.Sprintf("%+v", hdr))
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
	if slices.Contains(opts, ParseBodyOption) {
		sub = make([]byte, hdr.PayloadSize)
		n, err := r.ReadAt(sub, int64(hdr.PayloadOffset))
		if err != nil {
			return BootInfo{}, err
		}
		if n == 0 {
			return BootInfo{}, errors.New("invalid seek, zero")
		}
		if debug {
			debugStatement(fmt.Sprintf("read: %d", n))
		}
	}
	return BootInfo{header: hdr, body: sub}, nil
}

// Body will get the parsed body
func (info BootInfo) Body() []byte {
	return info.body
}

// Headers will get the underlying headers for the boot info
func (info BootInfo) Headers() Header {
	return info.header
}

// Unpack will write (and optionally decompress prior) the payload of the file
func Unpack(src Package, dst io.Writer, opts ...UnpackOption) error {
	if src == nil {
		return errors.New("unpacker is nil")
	}
	sub := src.Body()
	if len(sub) == 0 {
		return errors.New("no body")
	}
	debug := slices.Contains(opts, DebugUnpackOption)
	if slices.Contains(opts, DecompressOption) {
		found := false
		hdr := src.Headers()
		type decompressor struct {
			bodyType []uint8
			fxn      func([]byte) ([]byte, error)
		}
		for _, v := range []decompressor{
			{Gzip.Data(), decompressGunzip},
		} {
			if debug {
				debugStatement(fmt.Sprintf("compression: %v", v.bodyType))
			}
			if slices.Compare(hdr.CompressionType[:], v.bodyType) == 0 {
				found = true
				d, err := v.fxn(sub)
				if err != nil {
					return err
				}
				sub = d
			}
		}
		if !found {
			return fmt.Errorf("unknown compression type: %v", hdr.CompressionType[:])
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
				if debug {
					debugStatement(fmt.Sprintf("found type: %v", c.payload))
				}
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
	_, err := dst.Write(sub)
	return err
}
