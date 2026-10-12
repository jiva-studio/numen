package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// CreateAndImportURL is nil where the build has no way to make or fetch an address.
func (c Config) CreateAndImportURL(create *source.CreateURL, fetch *source.ImportURL) *source.CreateAndImportURL {
	if create == nil || fetch == nil {
		return nil
	}
	return source.NewCreateAndImportURL(create, fetch)
}
