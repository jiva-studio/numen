package indexport

// ModelRecorder keeps the name and the recipe of the model a vault is embedded under.
type ModelRecorder interface {
	RecordModel(name, recipe string)
}
