package engine

// Overrides is what the process environment sets for the model and the runtime; a field that is
// empty or zero is not set.
type Overrides struct {
	Model      string
	Repository string
	Dir        string
	File       string
	Pooling    string
	Runtime    string
	Dimensions int
	MaxTokens  int
	Threads    int
}

// ApplyToInfo sets the dimensions, the token limit and the pooling the overrides name.
func (o Overrides) ApplyToInfo(info *ModelInfo) {
	if o.Dimensions > 0 {
		info.Dimensions = o.Dimensions
	}
	if o.MaxTokens > 0 {
		info.MaxTokens = o.MaxTokens
	}
	if o.Pooling != "" {
		info.Pooling = o.Pooling
	}
}

// ApplyToLocal sets the repository, the directory, the file, the runtime and the threads the
// overrides name.
func (o Overrides) ApplyToLocal(local *LocalModel) {
	if o.Repository != "" {
		*local = local.WithRepository(o.Repository)
	}
	if o.Dir != "" {
		local.Dir = o.Dir
	}
	if o.File != "" {
		local.File = o.File
	}
	if o.Runtime != "" {
		local.Runtime = o.Runtime
	}
	if o.Threads > 0 {
		local.Threads = o.Threads
	}
}
