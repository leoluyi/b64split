# b64split

`b64split` streams one file into line-free Base64 text parts with a maximum size per part.

## Usage

```sh
go run . [--max-size 9MB] [--output-dir DIR] INPUT
```

The default part limit is 9MB, which means 9,000,000 bytes.

Use `9MiB` for 9 × 1,048,576 bytes, or pass a bare integer for a byte count.

The default output directory is `<input-file>.base64` beside the input file.

The command refuses to use an output directory that already exists.

Each part ends on a Base64 group boundary, so the parts can be concatenated and decoded as one stream.

On macOS, run `cat INPUT.base64/part-*.b64.txt | base64 -D > restored-file` to reconstruct the original.

On Linux, replace `base64 -D` with `base64 -d`.

## Install with Homebrew

```sh
brew tap leoluyi/homebrew-tap
brew trust --formula leoluyi/homebrew-tap/b64split
brew install b64split
```

## CI

Run `make ci` to check formatting, run `go vet` and `go test`, and build the CLI.

The individual checks are available as `make fmt-check`, `make vet`, `make test`, and `make build`.
