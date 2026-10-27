package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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
	name := fmt.Sprintf("part-%04d.b64.txt", len(w.paths)+1)
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
