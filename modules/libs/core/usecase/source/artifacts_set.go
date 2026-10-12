package source

// Artifacts is everything that acts on what was made from a vault's files. An
// agent's tools are served the set, and a tool is offered where its use case is
// here.
type Artifacts struct {
	List  *ListArtifacts
	Read  *ReadArtifact
	Write *WriteCorrections
}
