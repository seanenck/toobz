toobz
===

A Go implementation of un-zbooting an EFI zboot image (mainly for arm64).

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

To extract and decompress an image:
```
./toobz -in <image> -out <file> -decompress
```

## Reference

This is a Go re-implementation of [unzboot](https://github.com/eballetbo/unzboot) which is actually an
implementation derived from
[qemu](https://github.com/qemu/qemu/blob/master/hw/core/loader.c) both of which
are written in C
