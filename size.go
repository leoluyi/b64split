package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var sizePattern = regexp.MustCompile(`^([0-9]+)\s*([a-zA-Z]*)$`)

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
