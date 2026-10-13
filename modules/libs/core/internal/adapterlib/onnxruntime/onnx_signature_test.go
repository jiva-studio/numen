package onnxruntime_test

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// Element types of an ONNX tensor, by the number the file gives them.
const (
	elementFloat32 int32 = 1
	elementInt32   int32 = 6
	elementInt64   int32 = 7
	elementBool    int32 = 9
)

// dynamicDim is the extent a dimension takes in a tensorSignature when the
// model leaves it open.
const dynamicDim int64 = -1

// readLimit is how many bytes of a message are read into memory at once; a
// message longer than this is a weight, and only its leading fields are read.
const readLimit = 1 << 10

// A tensorSignature is one input or output of a model as the file declares it.
type tensorSignature struct {
	Name string
	Type int32
	// Dims holds dynamicDim for each dimension the model leaves open, and Params
	// the name the model gives each of them ("" for a fixed one).
	Dims   []int64
	Params []string
}

// A modelSignature is what a model takes and what it gives back.
type modelSignature struct {
	Inputs  []tensorSignature
	Outputs []tensorSignature
}

// readModelSignature reads the inputs and outputs a model declares, without
// reading its weights. An input that a weight of the model also names is not
// an input.
func readModelSignature(path string) (modelSignature, error) {
	file, err := os.Open(path)
	if err != nil {
		return modelSignature{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return modelSignature{}, err
	}

	var sig modelSignature
	weights := map[string]bool{}
	err = walkMessage(file, 0, info.Size(), func(field, kind int, at, size int64) error {
		if field != 7 || kind != wireBytes {
			return nil
		}
		return walkMessage(file, at, at+size, func(field, kind int, at, size int64) error {
			if kind != wireBytes {
				return nil
			}
			switch field {
			case 5:
				name, err := readLeadingName(file, at, size)
				if err != nil {
					return err
				}
				weights[name] = true
			case 11, 12:
				body, err := readAt(file, at, size)
				if err != nil {
					return err
				}
				tensor, err := parseValueInfo(body)
				if err != nil {
					return err
				}
				if field == 11 {
					sig.Inputs = append(sig.Inputs, tensor)
				} else {
					sig.Outputs = append(sig.Outputs, tensor)
				}
			}
			return nil
		})
	})
	if err != nil {
		return modelSignature{}, fmt.Errorf("read %s: %w", path, err)
	}

	kept := sig.Inputs[:0]
	for _, in := range sig.Inputs {
		if !weights[in.Name] {
			kept = append(kept, in)
		}
	}
	sig.Inputs = kept
	return sig, nil
}

const (
	wireVarint = 0
	wireBytes  = 2
)

// walkMessage calls visit for each field of the message in file[from:to]. A
// varint field is reported with its value in size.
func walkMessage(file io.ReaderAt, from, to int64, visit func(field, kind int, at, size int64) error) error {
	for at := from; at < to; {
		tag, n, err := readVarint(file, at)
		if err != nil {
			return err
		}
		at += n
		field, kind := int(tag>>3), int(tag&7)
		switch kind {
		case wireVarint:
			value, n, err := readVarint(file, at)
			if err != nil {
				return err
			}
			at += n
			if err := visit(field, kind, at, int64(value)); err != nil {
				return err
			}
		case wireBytes:
			size, n, err := readVarint(file, at)
			if err != nil {
				return err
			}
			at += n
			if err := visit(field, kind, at, int64(size)); err != nil {
				return err
			}
			at += int64(size)
		case 1:
			at += 8
		case 5:
			at += 4
		default:
			return fmt.Errorf("wire type %d at offset %d", kind, at)
		}
	}
	return nil
}

func readVarint(file io.ReaderAt, at int64) (uint64, int64, error) {
	var buf [binary.MaxVarintLen64]byte
	n, err := file.ReadAt(buf[:], at)
	if n == 0 && err != nil {
		return 0, 0, err
	}
	value, used := binary.Uvarint(buf[:n])
	if used <= 0 {
		return 0, 0, errors.New("malformed varint")
	}
	return value, int64(used), nil
}

func readAt(file io.ReaderAt, at, size int64) ([]byte, error) {
	body := make([]byte, size)
	if _, err := file.ReadAt(body, at); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return body, nil
}

// readLeadingName reads the name of a weight, which the file writes before the
// weight's data.
func readLeadingName(file io.ReaderAt, at, size int64) (string, error) {
	body, err := readAt(file, at, min(size, readLimit))
	if err != nil {
		return "", err
	}
	var name string
	_ = walkMessage(bytesAt(body), 0, int64(len(body)), func(field, kind int, at, size int64) error {
		if field == 8 && kind == wireBytes && at+size <= int64(len(body)) {
			name = string(body[at : at+size])
		}
		return nil
	})
	return name, nil
}

type bytesAt []byte

func (b bytesAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, io.EOF
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// parseValueInfo reads one ValueInfoProto: a name and a tensor type.
func parseValueInfo(body []byte) (tensorSignature, error) {
	var out tensorSignature
	err := walkMessage(bytesAt(body), 0, int64(len(body)), func(field, kind int, at, size int64) error {
		switch {
		case field == 1 && kind == wireBytes:
			out.Name = string(body[at : at+size])
		case field == 2 && kind == wireBytes:
			return parseTypeProto(body[at:at+size], &out)
		}
		return nil
	})
	return out, err
}

func parseTypeProto(body []byte, out *tensorSignature) error {
	return walkMessage(bytesAt(body), 0, int64(len(body)), func(field, kind int, at, size int64) error {
		if field != 1 || kind != wireBytes {
			return nil
		}
		tensor := body[at : at+size]
		return walkMessage(bytesAt(tensor), 0, int64(len(tensor)), func(field, kind int, at, size int64) error {
			switch {
			case field == 1 && kind == wireVarint:
				out.Type = int32(size)
			case field == 2 && kind == wireBytes:
				return parseShape(tensor[at:at+size], out)
			}
			return nil
		})
	})
}

func parseShape(body []byte, out *tensorSignature) error {
	return walkMessage(bytesAt(body), 0, int64(len(body)), func(field, kind int, at, size int64) error {
		if field != 1 || kind != wireBytes {
			return nil
		}
		dim := body[at : at+size]
		extent, param := dynamicDim, ""
		err := walkMessage(bytesAt(dim), 0, int64(len(dim)), func(field, kind int, at, size int64) error {
			switch {
			case field == 1 && kind == wireVarint:
				extent = size
			case field == 2 && kind == wireBytes:
				param = string(dim[at : at+size])
			}
			return nil
		})
		out.Dims = append(out.Dims, extent)
		out.Params = append(out.Params, param)
		return err
	})
}
