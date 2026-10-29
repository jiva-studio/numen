package domain

// Entry is one file or folder of a vault, as a listing of a folder reports it.
type Entry struct {
	// Path is relative to the vault root and always uses forward slashes, so a
	// listing names the same file on every platform.
	Path string
	// Name is the last segment of the path, as displayed to the user.
	Name string
	// IsFolder is whether a listing of this path has entries of its own.
	IsFolder bool
	// Kind is what the file is, and is empty where the vault holds no source at
	// the path: a folder, or a file nothing reads.
	Kind SourceKind
}
