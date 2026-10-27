package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultMaxSize = "9MB"

var sizePattern = regexp.MustCompile(`^([0-9]+)\s*([a-zA-Z]*)$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "b64split:", err)
		os.Exit(1)
	}
}

func run(args []string) (retErr error) {
	flags := flag.NewFlagSet("b64split", flag.ContinueOnError)
	maxSizeText := flags.String("max-size", defaultMaxSize, "maximum size of each Base64 part (default 9MB)")
	outputDirText := flags.String("output-dir", "", "directory for parts (default: <input-file>.base64)")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: b64split [flags] <input-file>")
		fmt.Fprintln(flags.Output(), "")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return errors.New("expected exactly one input file")
	}

	maxSize, err := parseSize(*maxSizeText)
	if err != nil {
		return fmt.Errorf("invalid --max-size: %w", err)
	}
	if maxSize < 4 {
		return errors.New("--max-size must be at least 4 bytes")
	}
	partLimit := maxSize - maxSize%4

	inputPath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("stat input file: %w", err)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("input is not a regular file: %s", inputPath)
	}
	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer input.Close()

	outputDir := *outputDirText
	if outputDir == "" {
		outputDir = inputPath + ".base64"
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputDir), 0o755); err != nil {
		return fmt.Errorf("create output parent directory: %w", err)
	}
	if err := os.Mkdir(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	parts := &partWriter{dir: outputDir, partLimit: partLimit}
	defer func() {
		if retErr != nil {
			parts.cleanup()
			_ = os.Remove(outputDir)
		}
	}()

	encoder := base64.NewEncoder(base64.StdEncoding, parts)
	if _, err := io.Copy(encoder, input); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("encode input: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finish Base64 encoding: %w", err)
	}
	if len(parts.paths) == 0 {
		if err := parts.openNext(); err != nil {
			return fmt.Errorf("create empty Base64 part: %w", err)
		}
	}
	if err := parts.closeCurrent(); err != nil {
		return fmt.Errorf("close output part: %w", err)
	}

	fmt.Printf("Wrote %d Base64 part(s) to %s (maximum %d bytes each).\n", len(parts.paths), outputDir, maxSize)
	return nil
}

func parseSize(value string) (int64, error) {
	matches := sizePattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return 0, fmt.Errorf("expected a byte count or a size such as 9MB or 9MiB")
	}

	amount, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %w", err)
	}
	unit := strings.ToUpper(matches[2])
	decimalUnits := map[string]int64{"": 1, "B": 1, "KB": 1_000, "MB": 1_000_000, "GB": 1_000_000_000, "TB": 1_000_000_000_000}
	binaryUnits := map[string]int64{"KIB": 1 << 10, "MIB": 1 << 20, "GIB": 1 << 30, "TIB": 1 << 40}
	multiplier, ok := decimalUnits[unit]
	if !ok {
		multiplier, ok = binaryUnits[unit]
	}
	if !ok {
		return 0, fmt.Errorf("unsupported size unit %q", matches[2])
	}
	if amount > math.MaxInt64/multiplier {
		return 0, errors.New("size is too large")
	}
	return amount * multiplier, nil
}

type partWriter struct {
	dir        string
	partLimit  int64
	current    *os.File
	currentLen int64
	paths      []string
}

func (w *partWriter) Write(data []byte) (int, error) {
	if len(data)%4 != 0 {
		return 0, errors.New("internal error: Base64 output is not aligned to 4-character groups")
	}

	total := 0
	for len(data) > 0 {
		if w.current == nil {
			if err := w.openNext(); err != nil {
				return total, err
			}
		}

		available := w.partLimit - w.currentLen
		count := len(data)
		if int64(count) > available {
			count = int(available)
		}
		count -= count % 4
		if count == 0 {
			return total, errors.New("internal error: no room for a Base64 group in the current part")
		}

		written, err := w.current.Write(data[:count])
		total += written
		w.currentLen += int64(written)
		data = data[written:]
		if err != nil {
			return total, err
		}
		if written != count {
			return total, io.ErrShortWrite
		}
		if w.currentLen == w.partLimit {
			if err := w.closeCurrent(); err != nil {
				return total, err
			}
		}
	}
	return total, nil
}

func (w *partWriter) openNext() error {
	name := fmt.Sprintf("part-%04d.b64", len(w.paths)+1)
	path := filepath.Join(w.dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	w.current = file
	w.currentLen = 0
	w.paths = append(w.paths, path)
	return nil
}

func (w *partWriter) closeCurrent() error {
	if w.current == nil {
		return nil
	}
	file := w.current
	w.current = nil
	w.currentLen = 0
	return file.Close()
}

func (w *partWriter) cleanup() {
	_ = w.closeCurrent()
	for _, path := range w.paths {
		_ = os.Remove(path)
	}
}
