package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "b64split")
	build := exec.Command("go", "build", "-race", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	for _, tc := range []struct {
		name      string
		data      []byte
		limit     string
		partLimit int
	}{
		{"empty", nil, "4", 4},
		{"padding", []byte("a"), "4", 4},
		{"multiple-parts", bytes.Repeat([]byte{0, 1, 127, 255}, 1024), "13", 12},
		{"binary-unit", []byte("hello world"), "1KiB", 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input")
			if err := os.WriteFile(input, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "--max-size", tc.limit, input)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("run CLI: %v\n%s", err, output)
			}
			parts, err := filepath.Glob(input + ".base64/part-*.b64.txt")
			if err != nil || len(parts) == 0 {
				t.Fatalf("find parts: %v, %v", parts, err)
			}
			var encoded []byte
			for _, part := range parts {
				data, err := os.ReadFile(part)
				if err != nil {
					t.Fatal(err)
				}
				if len(data) > tc.partLimit || len(data)%4 != 0 {
					t.Fatalf("invalid part size: %d", len(data))
				}
				encoded = append(encoded, data...)
			}
			decoded, err := base64.StdEncoding.Strict().DecodeString(string(encoded))
			if err != nil || !bytes.Equal(decoded, tc.data) {
				t.Fatalf("round trip failed: %v", err)
			}
			if output, err := exec.Command(binary, input).CombinedOutput(); err == nil {
				t.Fatalf("existing output directory accepted: %s", output)
			}
		})
	}

	for _, args := range [][]string{nil, {"--max-size", "invalid", "missing"}, {"--max-size", "3", "missing"}, {"missing"}} {
		if output, err := exec.Command(binary, args...).CombinedOutput(); err == nil {
			t.Errorf("expected failure for %v: %s", args, output)
		}
	}
	if output, err := exec.Command(binary, "--help").CombinedOutput(); err != nil {
		t.Fatalf("help failed: %v\n%s", err, output)
	}
}
