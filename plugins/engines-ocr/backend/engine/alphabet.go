package engine

import (
	"encoding/binary"
	"errors"
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

// alphabet returns the characters the model reads, from the dictionary named in
// the settings or else from the model's metadata, in the format the recogniser
// takes: one per line, with the space last. The model supplies the characters,
// the space, and the CTC blank.
func alphabet(model string, cfg RecogniserModel) (int64, string, error) {
	characters, err := readAlphabet(model, cfg.Dict)
	if err != nil {
		return 0, "", err
	}
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(characters, "\r\n", "\n"), "\n"), "\n")

	classes := int64(len(lines)) + 2
	lines = append(lines, " ")
	if cfg.Classes > 0 && cfg.Classes != classes {
		return 0, "", fmt.Errorf("%s has %d characters but the settings specify %d", model, classes, cfg.Classes)
	}

	file, err := os.CreateTemp("", "numen-alphabet-*.txt")
	if err != nil {
		return 0, "", err
	}
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		return 0, "", errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return 0, "", err
	}
	return classes, file.Name(), nil
}

func readMetadata(path, key string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for pos := 0; pos < len(raw); {
		field, wire, size, ok := parseTag(raw[pos:])
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		pos += size

		body, size, ok := readPayload(raw[pos:], wire)
		if !ok {
			return "", fmt.Errorf("%s is not a model this can read", path)
		}
		pos += size

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
	for pos := 0; pos < len(raw); {
		field, wire, size, good := parseTag(raw[pos:])
		if !good {
			return "", "", false
		}
		pos += size
		body, size, good := readPayload(raw[pos:], wire)
		if !good {
			return "", "", false
		}
		pos += size

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

// readAlphabet reads a PaddleOCR character dictionary, UTF-8 with one character per line, from the
// file named, or else from the model's metadata.
func readAlphabet(model, dict string) (string, error) {
	if dict != "" {
		raw, err := os.ReadFile(dict)
		if err != nil {
			return "", fmt.Errorf("the character dictionary: %w", err)
		}
		return string(raw), nil
	}
	characters, err := readMetadata(model, alphabetKey)
	if err != nil {
		return "", fmt.Errorf("%s carries no alphabet, and none was configured: %w", model, err)
	}
	return characters, nil
}
