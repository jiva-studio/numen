package vaultport

// WriteAdmission registers an operation with its caller. An admitted operation
// holds its registration through the final index update and then releases it.
type WriteAdmission func() (release func(), err error)

// Begin admits the write. A nil policy admits writes without tracking them.
func (admit WriteAdmission) Begin() (func(), error) {
	if admit == nil {
		return func() {}, nil
	}
	return admit()
}
