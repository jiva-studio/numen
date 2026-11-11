package plugin

import "github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"

// SetStorage sets the folders the plugins store files in. Where the storage names a models folder, the
// dependencies of the plugins are provisioned into it by the registry newRegistry makes.
func (h *Host) SetStorage(cacheRoot, dataRoot string, newRegistry func(modelsDir string) pluginport.DependencyRegistry) {
	h.Storage.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line reason="a window started without a models directory gets no dependency registry, which only an installation with downloaded models exercises"
	if models := h.Storage.GetModelsDir(); models != "" {
		// gomutants:disable-next-line reason="the registry is built from the models directory of an installation, which a test does not have"
		h.Services.SetDependencies(newRegistry(models))
	}
}
