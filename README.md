# b64split

`b64split` streams a file into standard Base64 text parts, each bounded by a configurable size limit.
It is useful when an encoded file must fit within per-file upload or transfer limits.
The input is read incrementally, and output contains no line breaks.

## Installation

### Homebrew

```sh
brew tap leoluyi/homebrew-tap
brew trust --formula leoluyi/homebrew-tap/b64split
brew install b64split
```

### Build from source

Development requires Go 1.20 or newer and Make.
The project uses only the Go standard library.

```sh
git clone https://github.com/leoluyi/b64split.git
cd b64split
make build
./bin/b64split --help
```

`make build` retains the executable at `bin/b64split` for local use.
You can also run the CLI from the repository with `go run .`.

## Usage

```text
b64split [--max-size SIZE] [--output-dir DIR] INPUT
```

Place flags before the input path.
Exactly one regular input file is required; directories and standard input are not supported.

```sh
# Use the default 9MB part limit.
b64split archive.zip

# Use a binary size unit and a custom output directory.
b64split --max-size 9MiB --output-dir ./parts archive.zip

# Run directly from source.
go run . --max-size 1000000 archive.zip
```

| Option | Default | Meaning |
| --- | --- | --- |
| `--max-size SIZE` | `9MB` | Maximum encoded size of each part, in bytes |
| `--output-dir DIR` | `<input-file>.base64` beside the input | New directory to hold the parts |
| `--help` | | Print usage and exit successfully |

Size values accept nonnegative integers with case-insensitive units.
Fractional values such as `1.5MB` are not accepted.

| Unit | Multiplier |
| --- | --- |
| No suffix or `B` | 1 |
| `KB`, `MB`, `GB`, `TB` | 1,000 to the corresponding power |
| `KiB`, `MiB`, `GiB`, `TiB` | 1,024 to the corresponding power |

For example, `9MB` is 9,000,000 bytes and `9MiB` is 9,437,184 bytes.
The minimum limit is 4 bytes.
Limits are rounded down to a multiple of four so each part ends on a complete Base64 group; `--max-size 13` therefore produces parts no larger than 12 bytes.
The limit applies to encoded output, which is roughly one third larger than the original data.

## Output and restoration

A default invocation creates files such as:

```text
archive.zip.base64/
├── part-0001.b64.txt
├── part-0002.b64.txt
└── part-0003.b64.txt
```

Part numbering starts at 1, with at least four digits.
An empty input produces one empty part.
The output directory must not already exist, even if it is empty; missing parent directories are created automatically.
The input file is preserved.
On an encoding or output error, the command attempts to remove the parts and output directory it created.
Successful execution exits with status 0; invalid arguments and processing failures exit with status 1 and print an error to stderr.

To restore the original file, concatenate parts in numeric order and decode the combined stream.
This example sorts numerically even when there are more than 9,999 parts:

```sh
# macOS
(
  cd archive.zip.base64 || exit 1
  printf '%s\n' part-*.b64.txt | sort -t- -k2,2n |
    while IFS= read -r part; do cat "$part"; done
) | /usr/bin/base64 -D > restored.zip

cmp archive.zip restored.zip
```

The macOS example explicitly uses the system decoder at `/usr/bin/base64`.
On Linux or when using GNU Coreutils, replace `/usr/bin/base64 -D` with `base64 -d`.
Use a different destination path from the original input when restoring.

## Development

The Makefile is the shared entry point for local work and GitHub Actions.
Tests with the race detector require a working C compiler and a Go platform that supports the race detector.

| Target | Purpose |
| --- | --- |
| `make` or `make ci` | Check formatting, vet, test, and verify a build in an automatically cleaned temporary directory |
| `make fmt` | Format Go source files |
| `make fmt-check` | Check formatting without changing files |
| `make vet` | Run static checks |
| `make test` | Run tests with the race detector, bypassing cached test results |
| `make build` | Build and retain `bin/b64split` |
| `make dist VERSION=v1.0.0` | Package release archives and SHA-256 checksums in `dist/` |
| `make release VERSION=v1.0.0` | Validate, package, and publish a GitHub release |
| `make repo-path` | Print the repository directory |

CLI integration tests build and execute the program to check empty input, Base64 padding, multiple parts, size limits, restoration, existing output directories, invalid arguments, and help.
`make ci` removes its temporary build directory on success or failure and does not retain a binary in the repository.
`make dist` also stages its build in a temporary directory, retaining only the archives and checksum file in `dist/`.
The `bin/` and `dist/` directories are ignored by Git.

The following variables can be overridden:

| Variable | Default | Purpose |
| --- | --- | --- |
| `GO` | `go` | Go command used by Makefile recipes |
| `GOFMT` | `gofmt` | Formatter command |
| `BINARY` | `bin/b64split` | Standalone `make build` output path; CI supplies its own temporary path |
| `DIST_DIR` | `dist` | Directory for packaged outputs |
| `VERSION` | `dev` | Version label included in archive names; an existing tag is required for publishing |
| `PLATFORMS` | `darwin/amd64 darwin/arm64 linux/amd64 linux/arm64` | Space-separated packaging targets |

```sh
make ci
make build BINARY=bin/custom-name
make dist VERSION=v1.0.0 PLATFORMS='linux/amd64 linux/arm64'
```

## CI and releases

GitHub Actions runs `make ci` on pull requests, pushes to `main`, matching version tags, and manual workflow runs.
The CI matrix covers Linux and macOS with Go 1.20 and the current stable Go release.
Check jobs have read-only repository contents permission.

Pushing a tag matching `v[0-9]*`, such as `v1.0.0`, triggers checks followed by a release job that invokes `make release`.
Release tags must contain only letters, digits, dots, underscores, or hyphens after the initial `v` and digit.
The release job has repository contents write permission and publishes a GitHub release with generated notes.
Homebrew tap updates remain manual.

For each platform, `make dist VERSION=v1.0.0` produces an archive such as `b64split_v1.0.0_darwin_arm64.tar.gz` containing the executable and this README.
A shared `checksums.txt` lists SHA-256 checksums for the archives.
Verify downloaded archives from their containing directory with:

```sh
shasum -a 256 -c checksums.txt
```

That command expects every archive listed in `checksums.txt` to be present.

To publish manually, first ensure the intended version tag exists locally, points to `HEAD`, and has been pushed to GitHub.
Use a clean checkout of the tagged source so the archives match the tag.
An authenticated GitHub CLI with repository write access is required.

```sh
make release VERSION=v1.0.0
```

Publishing an already existing release fails rather than replacing its assets.

## Project layout

```text
.github/workflows/ci.yml   CI matrix and tag release automation
cli.go                     Flags, validation, and encoding workflow
cli_test.go                Executable CLI integration tests
go.mod                     Go module and minimum language version
main.go                    Process entry point and exit handling
Makefile                   Validation, builds, packaging, and publishing
part_writer.go             Output part rotation and cleanup
size.go                    Byte size parsing
```

The CLI stays in the root package, so `go run .` and `go build .` work from the repository root.
