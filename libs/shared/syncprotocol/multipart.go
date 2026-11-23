package syncprotocol

import "errors"

// ErrInvalidPartSize means a size and a part size do not describe a multipart upload the protocol allows.
var ErrInvalidPartSize = errors.New("the size and the part size do not fit a multipart upload")

// CheckReserveMultipart returns ErrInvalidPartSize unless a body of the stored size can be sent in parts of the
// part size.
func CheckReserveMultipart(size, partSize uint64) error {
	if size <= MaxBlobSize || size > MaxMultipartSize {
		return ErrInvalidPartSize
	}
	if partSize < MinPartSize || partSize > MaxPartSize {
		return ErrInvalidPartSize
	}
	if (size+partSize-1)/partSize > MaxParts {
		return ErrInvalidPartSize
	}
	return nil
}
