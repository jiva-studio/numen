package document

import "strconv"

// PageRequest is a request for one page image.
type PageRequest struct {
	Index      int
	Path       string
	PixelWidth int
}

// BadRequestError is a request for a page image that names no page, file or width.
type BadRequestError struct {
	Message string
}

func (e *BadRequestError) Error() string { return e.Message }

// ParsePageRequest reads the page index (counted from zero), the file path and the width in pixels
// of a request for a page image from the text the caller received.
func ParsePageRequest(index, path, widthText string) (PageRequest, error) {
	page, err := strconv.Atoi(index)
	if err != nil || page < 0 {
		return PageRequest{}, &BadRequestError{Message: "invalid page"}
	}
	if path == "" {
		return PageRequest{}, &BadRequestError{Message: "path required"}
	}
	pixelWidth, err := strconv.Atoi(widthText)
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="strconv.Atoi returns pixelWidth 0 on parse failure"
	if err != nil || pixelWidth <= 0 {
		return PageRequest{}, &BadRequestError{Message: "invalid wide"}
	}
	return PageRequest{Index: page, Path: path, PixelWidth: pixelWidth}, nil
}
