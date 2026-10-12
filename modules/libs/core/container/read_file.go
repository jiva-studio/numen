package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/file"

func (c Config) Read() file.Read {
	return file.Read{Readers: c.VaultReaders()}
}
