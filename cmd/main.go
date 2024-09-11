// Package main is a CLI wrapper for unpacking
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
	debug := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()
	debugging := *debug
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
	readOpts := []toobz.ReadInfoOption{toobz.ParseBodyOption}
	if debugging {
		readOpts = append(readOpts, toobz.DebugReadInfoOption)
	}
	info, err := toobz.ReadInfo(buf, readOpts...)
	if err != nil {
		return err
	}
	w, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer w.Close()
	unpackOpts := []toobz.UnpackOption{}
	if *decompress {
		unpackOpts = append(unpackOpts, toobz.DecompressOption)
	}
	if debugging {
		unpackOpts = append(unpackOpts, toobz.DebugUnpackOption)
	}
	return toobz.Unpack(info, w, unpackOpts...)
}
