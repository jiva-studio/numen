package settings

// Bounds is how far a setting holding a number goes, at each end.
type Bounds struct{ Least, Most float64 }

// Current is the settings file as JSON and at its current position, with the models and
// limits that go with the settings.
type Current struct {
	Written                  string
	Path                     string
	Models                   []Model
	NestedHeadingLimitBounds Bounds
	Plugins                  []PluginInfo
}

// Get reads every setting, the models a setting can name, and the limits the
// settings are refused outside.
type Get struct {
	read                     func() (written, path string, err error)
	models                   func() []Model
	nestedHeadingLimitBounds Bounds
	plugins                  func() []PluginInfo
}

func NewGet(
	read func() (written, path string, err error),
	models func() []Model,
	nestedHeadingLimitBounds Bounds,
	plugins func() []PluginInfo,
) *Get {
	return &Get{
		read:                     read,
		models:                   models,
		nestedHeadingLimitBounds: nestedHeadingLimitBounds,
		plugins:                  plugins,
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
		Written:                  content,
		Path:                     path,
		Models:                   models,
		NestedHeadingLimitBounds: u.nestedHeadingLimitBounds,
		Plugins:                  pluginInfos,
	}, nil
}
