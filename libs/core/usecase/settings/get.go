package settings

// Current is the settings file as JSON and at its current position, with the models and
// plugins that go with the settings.
type Current struct {
	Written string
	Path    string
	Models  []Model
	Plugins []PluginInfo
}

// Get reads every setting, the models a setting can name, and the limits the
type Get struct {
	read    func() (content, path string, err error)
	models  func() []Model
	plugins func() []PluginInfo
}

func NewGet(
	read func() (content, path string, err error),
	models func() []Model,
	plugins func() []PluginInfo,
) *Get {
	return &Get{
		read:    read,
		models:  models,
		plugins: plugins,
	}
}

func (u *Get) Execute() (Current, error) {
	content, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	var models []Model
	if u.models != nil {
		models = u.models()
	}
	var pluginInfos []PluginInfo
	if u.plugins != nil {
		pluginInfos = u.plugins()
	}
	return Current{
		Written: content,
		Path:    path,
		Models:  models,
		Plugins: pluginInfos,
	}, nil
}
