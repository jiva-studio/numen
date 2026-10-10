package embed

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The shelves the models stand on.
const (
	shelfMachine    = "On this machine"
	shelfConfigured = "Named in the settings"
)

// ModelAt is where the model a vault is indexed by stands inside this adapter's
// section of the settings.
var ModelAt = []string{"model", "name"}

// Profiles offers pre-quantized and hardware-accelerated model presets (e.g. INT8 for VNNI / Apple Neural Engine).
func Profiles() []Config {
	return []Config{
		Defaults(),
		{
			Model: Model{Name: "e5-small-int8", Dimensions: 384, MaxTokens: 256, Pooling: PoolMean},
			Indexing: Provider{
				Use:   UseLocal,
				local: LocalModel{Name: "intfloat/multilingual-e5-small", File: "model_qint8_avx512_vnni.onnx", BatchTexts: 32, ShouldDownload: true},
			},
		},
		{
			Model: Model{Name: "bge-m3-int8", Dimensions: 1024, MaxTokens: 8192, Pooling: PoolHead},
			Indexing: Provider{
				Use:   UseLocal,
				local: LocalModel{Name: "Xenova/bge-m3", File: "model_int8.onnx", BatchTexts: 32, ShouldDownload: true},
			},
		},
		{
			Model: Model{Name: "all-minilm-l6-v2-int8", Dimensions: 384, MaxTokens: 256, Pooling: PoolMean},
			Indexing: Provider{
				Use:   UseLocal,
				local: LocalModel{Name: "Xenova/all-MiniLM-L6-v2", File: "model_int8.onnx", BatchTexts: 32, ShouldDownload: true},
			},
		},
	}
}

// GetModels are the models a vault can be indexed by, read against the settings
// in force: each row says what its files are on this machine, and a model the
// settings name that is none of them stands as a row of its own.
//
// The name is what a vector is kept under, and the width, the window and the
// pooling belong with it, so choosing one writes the model and the provider
// that runs it together.
//
// Every place named here stands inside this section, and whoever composes the
// file says where the section is.
func GetModels(held Config, isFetched func(LocalModel) bool) []port.Model {
	var models []port.Model
	offered := Defaults()
	provider := held.Indexing

	seenNames := make(map[string]bool)

	for _, p := range Profiles() {
		local, hasLocal := p.Indexing.Local()
		var writes []port.Setting
		writes = append(writes,
			newSetting([]string{"model", "name"}, p.Model.Name),
			newSetting([]string{"model", "dimensions"}, p.Model.Dimensions),
			newSetting([]string{"model", "max_tokens"}, p.Model.MaxTokens),
			newSetting([]string{"model", "pooling"}, p.Model.Pooling),
			newSetting([]string{"indexing", "use"}, p.Indexing.Use),
		)
		if hasLocal {
			writes = append(writes, newSetting([]string{"indexing", "local", "name"}, local.Name))
			if local.File != "" {
				writes = append(writes, newSetting([]string{"indexing", "local", "file"}, local.File))
			}
		}

		var presence port.Presence
		if providerLocal, ok := provider.Local(); ok {
			if hasLocal {
				providerLocal.Name = local.Name
				providerLocal.File = local.File
			} else {
				providerLocal.Name = p.Model.Name
			}
			if isFetched(providerLocal) {
				presence = port.Present
			} else {
				presence = port.NotFetched
			}
		} else {
			presence = port.NothingToFetch
		}

		models = append(models, port.Model{
			Path:      ModelAt,
			Name:      p.Model.Name,
			Title:     p.Model.Name,
			Shelf:     shelfMachine,
			IsDefault: p.Model.Name == offered.Model.Name,
			Presence:  presence,
			Writes:    writes,
		})
		seenNames[p.Model.Name] = true
	}

	name := held.Model.Name
	if name == "" || seenNames[name] {
		return models
	}

	writes := []port.Setting{
		newSetting([]string{"model", "name"}, held.Model.Name),
		newSetting([]string{"model", "dimensions"}, held.Model.Dimensions),
		newSetting([]string{"model", "max_tokens"}, held.Model.MaxTokens),
		newSetting([]string{"model", "pooling"}, held.Model.Pooling),
		newSetting([]string{"indexing", "use"}, provider.Use),
	}
	if local, ok := provider.Local(); ok {
		writes = append(writes, newSetting([]string{"indexing", "local", "name"}, local.Name))
		if local.File != "" {
			writes = append(writes, newSetting([]string{"indexing", "local", "file"}, local.File))
		}
	}

	models = append(models, port.Model{
		Path:     ModelAt,
		Name:     name,
		Title:    name,
		Shelf:    shelfConfigured,
		Presence: getPresence(provider, name, isFetched),
		Writes:   writes,
	})

	return models
}

// getPresence is what the model named is on this machine, at the provider the
// settings run it at. A provider reaching a service fetches nothing whatever
// the model is called.
func getPresence(provider Provider, name string, isFetched func(LocalModel) bool) port.Presence {
	local, here := provider.Local()
	if !here {
		return port.NothingToFetch
	}
	local.Name = name
	if isFetched(local) {
		return port.Present
	}
	return port.NotFetched
}

// newSetting is one setting written down.
func newSetting(at []string, value any) port.Setting {
	said, err := json.Marshal(value)
	if err != nil {
		return port.Setting{Path: at, JSON: ""}
	}
	return port.Setting{Path: at, JSON: string(said)}
}
