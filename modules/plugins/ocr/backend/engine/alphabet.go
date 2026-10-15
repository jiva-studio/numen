package engine

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

const alphabetKey = "character"

const (
	metadataField = 14
	keyField      = 1
	valueField    = 2
)

func alphabet(model string, cfg RecogniserModel) (int64, string, error) {
	if cfg.Dict != "" {
		if cfg.Classes <= 0 {
			return 0, "", fmt.Errorf("a dictionary was named without saying how many characters the model knows")
		}
		return cfg.Classes, cfg.Dict, nil
	}

	characters, err := readMetadata(model, alphabetKey)
	if err != nil {
		return 0, "", fmt.Errorf("%s carries no alphabet, and none was configured: %w", model, err)
	}
	lines := strings.Split(strings.TrimRight(characters, "\n"), "\n")

	classes := int64(len(lines)) + 2
	lines = append(lines, " ")
	if cfg.Classes > 0 && cfg.Classes != classes {
		return 0, "", fmt.Errorf("%s knows %d characters and the settings say %d", model, classes, cfg.Classes)
	}

	file, err := os.CreateTemp("", "numen-alphabet-*.txt")
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		return 0, "", err
	}
	return classes, file.Name(), nil
}

func readMetadata(path, key string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for at := 0; at < len(raw); {
		field, wire, size, ok := parseTag(raw[at:])
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		at += size

		body, size, ok := readPayload(raw[at:], wire)
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		at += size

		if field != metadataField {
			continue
		}
		if k, v, ok := parseEntry(body); ok && k == key {
			return v, nil
		}
	}
	return "", fmt.Errorf("no %q in the model's metadata", key)
}

func parseEntry(raw []byte) (key, value string, ok bool) {
	for at := 0; at < len(raw); {
		field, wire, size, good := parseTag(raw[at:])
		if !good {
			return "", "", false
		}
		at += size
		body, size, good := readPayload(raw[at:], wire)
		if !good {
			return "", "", false
		}
		at += size

		switch field {
		case keyField:
			key = string(body)
		case valueField:
			value = string(body)
		}
	}
	return key, value, key != ""
}

func parseTag(raw []byte) (field int, wire byte, size int, ok bool) {
	n, size := binary.Uvarint(raw)
	if size <= 0 {
		return 0, 0, 0, false
	}
	return int(n >> 3), byte(n & 7), size, true
}

func readPayload(raw []byte, wire byte) (body []byte, size int, ok bool) {
	switch wire {
	case 0:
		_, n := binary.Uvarint(raw)
		if n <= 0 {
			return nil, 0, false
		}
		return nil, n, true
	case 1:
		if len(raw) < 8 {
			return nil, 0, false
		}
		return nil, 8, true
	case 2:
		length, n := binary.Uvarint(raw)
		if n <= 0 || uint64(len(raw)-n) < length {
			return nil, 0, false
		}
		return raw[n : n+int(length)], n + int(length), true
	case 5:
		if len(raw) < 4 {
			return nil, 0, false
		}
		return nil, 4, true
	}
	return nil, 0, false
}
