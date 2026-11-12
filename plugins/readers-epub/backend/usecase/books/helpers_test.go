package books

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

func codeOf(err error) server.ErrorCode {
	if failure, ok := errors.AsType[*server.APIError](err); ok {
		return failure.Code
	}
	return ""
}
