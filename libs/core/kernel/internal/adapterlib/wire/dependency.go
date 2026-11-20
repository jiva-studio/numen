package wire

import v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

func DescribeModelFile(path string, err error) *v1.FetchModelFileResponse {
	if err != nil {
		return &v1.FetchModelFileResponse{Error: err.Error()}
	}
	return &v1.FetchModelFileResponse{Path: path}
}
