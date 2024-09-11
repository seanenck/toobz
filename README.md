toobz
===

A Go implementation of un-zbooting an EFI zboot image (mainly for arm64).

## Why?

Run into an image with `file` output like
```
PE32+ executable (EFI application) Aarch64 (stripped to external PDB), for MS Windows
```

or?
```
PE32+ executable (EFI application) RISC-V 64-bit (stripped to external PDB), for MS Windows
```

These files need to be unpacked in a few cases where the raw kernel is desired.
There are other formats and types of similar looking files (e.g. sectioned) that
`toobz` does not currently support.

## Build

Clone and run
```
make
```

## Usage

To extract an image
```
toobz -in <image> -out <file>
```

To extract and decompress an image:
```
toobz -in <image> -out <file> -decompress
```

## Reference

This is a Go re-implementation of [unzboot](https://github.com/eballetbo/unzboot) which is actually an
implementation derived from
[qemu](https://github.com/qemu/qemu/blob/master/hw/core/loader.c) both of which
are written in C.
