// Package main unpacks an EFI zboot file
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/seanenck/toobz"
)

func main() {
	if err := unpack(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func unpack() error {
	in := flag.String("in", "", "input file")
	out := flag.String("out", "", "output file")
	decompress := flag.Bool("decompress", false, "decompress the payload")
	flag.Parse()
	unpacker := toobz.Unpacker{}
	unpacker.Decompress = *decompress
	unpacker.ParseBody = true
	input := *in
	output := *out
	if input == "" || output == "" {
		return errors.New("input/output must be defined")
	}
	b, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(b)
	info, err := unpacker.ReadInfo(buf)
	if err != nil {
		return err
	}
	w, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer w.Close()
	return info.Write(w)
}
