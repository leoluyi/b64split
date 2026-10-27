package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const defaultMaxSize = "9MB"

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
