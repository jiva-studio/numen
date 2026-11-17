package books

import "errors"

func codeOf(err error) Kind {
	if failure, ok := errors.AsType[*Error](err); ok {
		return failure.Kind
	}
	return 0
}
