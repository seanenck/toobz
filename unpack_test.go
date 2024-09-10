package toobz_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/seanenck/toobz"
)

var testData = "TVoAAHppbWdIyQAA0tiyAAAAAAAAAAAAZ3ppcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADNI4KBQAAAAA=="

func getTestData() []byte {
	n, _ := base64.StdEncoding.DecodeString(testData)
	return n
}

func TestDatum(t *testing.T) {
	a := toobz.ARM
	if a.Value() != "ARM" {
		t.Error("invalid payload value")
	}
	if fmt.Sprintf("%v", a.Data()) != "[65 82 77 100]" {
		t.Errorf("invalid payload data: %v", a.Data())
	}
	a = toobz.MSDOSMagic
	if a.Value() != "MZ" {
		t.Error("invalid payload value")
	}
	if fmt.Sprintf("%v", a.Data()) != "[77 90]" {
		t.Errorf("invalid payload data: %v", a.Data())
	}
	a = toobz.Gzip
	if a.Value() != "gzip" {
		t.Error("invalid payload value")
	}
	if fmt.Sprintf("%v", a.Data()) != "[103 122 105 112 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]" {
		t.Errorf("invalid payload data: %v", a.Data())
	}
	a = toobz.LinuxMagic
	if a.Value() != "Linux" {
		t.Error("invalid payload value")
	}
	if fmt.Sprintf("%v", a.Data()) != "[205 35 130 129]" {
		t.Errorf("invalid payload data: %v", a.Data())
	}
}

func TestReadInfo(t *testing.T) {
	u := toobz.Unpacker{}
	if _, err := u.ReadInfo(nil); err == nil || err.Error() != "reader is nil" {
		t.Errorf("invalid read info error: %v", err)
	}
	var data []byte
	if _, err := u.ReadInfo(bytes.NewReader(data)); err == nil || err.Error() != "EOF" {
		t.Errorf("invalid read info error, bad data: %v", err)
	}
	data = getTestData()
	data[0] = 1
	if _, err := u.ReadInfo(bytes.NewReader(data)); err == nil || !strings.Contains(err.Error(), "invalid content data") || !strings.Contains(err.Error(), "MZ") {
		t.Errorf("invalid read info error, bad data header: %v", err)
	}
	data = getTestData()
	data[8] = 0
	data[9] = 0
	data[10] = 0
	data[11] = 0
	if _, err := u.ReadInfo(bytes.NewReader(data)); err == nil || err.Error() != "payload size/offset is zero" {
		t.Errorf("invalid read info error, bad data payload size/offset: %v", err)
	}
	data = getTestData()
	data[12] = 0
	data[13] = 0
	data[14] = 0
	data[15] = 0
	if _, err := u.ReadInfo(bytes.NewReader(data)); err == nil || err.Error() != "payload size/offset is zero" {
		t.Errorf("invalid read info error, bad data payload size/offset: %v", err)
	}
	data = getTestData()
	data[8] = 255
	data[9] = 255
	data[10] = 255
	data[11] = 255
	data[12] = 255
	data[13] = 255
	data[14] = 255
	data[15] = 255
	if _, err := u.ReadInfo(bytes.NewReader(data)); err == nil || err.Error() != "invalid offset/payload, beyond size" {
		t.Errorf("invalid read info error, bad data sizing: %v", err)
	}
	data = getTestData()
	data[12] = 100
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i := 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	if _, err := u.ReadInfo(bytes.NewReader(data)); err != nil {
		t.Errorf("invalid error: %v", err)
	}
}

func TestReadInfoBody(t *testing.T) {
	u := toobz.Unpacker{}
	u.ParseBody = true
	data := getTestData()
	data = getTestData()
	data[12] = 1
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i := 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	b, err := u.ReadInfo(bytes.NewReader(data))
	if err != nil {
		t.Errorf("invalid info read: %v", err)
	}
	if len(b.Body()) == 0 {
		t.Errorf("invalid info read, no body: %v", err)
	}
}

func TestWrite(t *testing.T) {
	info := toobz.BootInfo{}
	var b bytes.Buffer
	if err := info.Write(&b); err == nil || err.Error() != "no body" {
		t.Errorf("invalid write: %v", err)
	}
	u := toobz.Unpacker{}
	u.ParseBody = true
	data := getTestData()
	data = getTestData()
	data[12] = 1
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i := 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	read, _ := u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err != nil {
		t.Errorf("invalid write: %v", err)
	}
	if b.Len() == 0 {
		t.Error("invalid write")
	}
}

func TestWriteDecompress(t *testing.T) {
	var b bytes.Buffer
	u := toobz.Unpacker{}
	u.ParseBody = true
	u.Decompress = true
	data := getTestData()
	data = getTestData()
	data[12] = 1
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i := 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	read, _ := u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err == nil || err.Error() != "unexpected EOF" {
		t.Errorf("invalid write: %v", err)
	}
	data = getTestData()
	data[12] = 1
	data[13] = 0
	data[14] = 0
	data[15] = 0
	data[24] = 0
	i = 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	read, _ = u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err == nil || !strings.Contains(err.Error(), "unknown compression type") {
		t.Errorf("invalid write, compression: %v", err)
	}
	data = getTestData()
	data[12] = 10
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i = 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	i = 51528
	j := 0
	g := gzipData()
	for j < len(g) {
		data[i] = g[j]
		j++
	}
	read, _ = u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err == nil || !strings.Contains(err.Error(), "gzip: invalid header") {
		t.Errorf("invalid write, compression: %v", err)
	}
}

func gzipData(s ...string) []byte {
	var w bytes.Buffer
	r := gzip.NewWriter(&w)
	args := s
	if len(args) == 0 {
		args = append(args, "this is a test string")
	}
	for _, k := range args {
		r.Write([]byte(k))
	}
	r.Close()
	return w.Bytes()
}

func TestWriteDecompressCheck(t *testing.T) {
	var b bytes.Buffer
	u := toobz.Unpacker{}
	u.ParseBody = true
	u.Decompress = true
	data := getTestData()
	data[12] = 45
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i := 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	i = 51528
	j := 0
	g := gzipData()
	for j < len(g) {
		data[i] = g[j]
		i++
		j++
	}
	read, _ := u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err == nil || !strings.Contains(err.Error(), "invalid response payload: 21") {
		t.Errorf("invalid write, compression: %v", err)
	}
	b = bytes.Buffer{}
	data = getTestData()
	data[12] = 90
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i = 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	i = 51528
	j = 0
	g = gzipData("ARMt1oij12j01j201 aifj109J0(#J", "a98hj98JP( *H#(*HQ(!", "OIJEOSJOSJ:OJI:Q")
	for j < len(g) {
		data[i] = g[j]
		i++
		j++
	}
	read, _ = u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err == nil || !strings.Contains(err.Error(), "unknown payload type: [74 79 83 74]") {
		t.Errorf("invalid write, payload: %v", err)
	}
	b = bytes.Buffer{}
	data = getTestData()
	data[12] = 91
	data[13] = 0
	data[14] = 0
	data[15] = 0
	i = 0
	for i < 65535 {
		data = append(data, byte(i))
		i += 1
	}
	i = 51528
	j = 0
	g = gzipData("ARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMdARMd",
		"1oi2j198j()*J(D*J#!J!)(!J )(J@! J@!)(J)!(@J){(D{J )(C*J{ )*(J{@")
	for j < len(g) {
		data[i] = g[j]
		i++
		j++
	}
	read, _ = u.ReadInfo(bytes.NewReader(data))
	b = bytes.Buffer{}
	if err := read.Write(&b); err != nil {
		t.Errorf("invalid write: %v", err)
	}
}
