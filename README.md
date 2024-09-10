toobz
===

A Go implementation of un-zbooting an EFI zboot image (mainly for arm64).
Unlike similar tools, it does NOT attempt to decompress the resulting image
(e.g. a gzip file) and instead leaves that to the user/common tooling (e.g. 
`gzip -d <file>`) and simply writes the compressed payload to the specified
output path.

## Build

Clone and run
```
go build toobz.go
```

## Usage

To extract an image
```
./toobz -in <image> -out <file>
```

(or just do `go run toobz.go ...`)

## Reference

This is a Go re-implementation of [unzboot](https://github.com/eballetbo/unzboot) which is actually an
implementation derived from
[qemu](https://github.com/qemu/qemu/blob/master/hw/core/loader.c) both of which
are written in C
