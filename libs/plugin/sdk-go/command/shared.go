package command

// SharedCommand is a shared command of the application, as libs/core/protocol/commands.json declares it.
// It has the same entries as SHARED_COMMANDS of the TypeScript SDK.
//
//numen:api
type SharedCommand struct {
	// ID is the full id of the command, such as app:file.createText.
	ID string
	// Title is what a person reads the command as.
	Title string
	// Arguments are the arguments the command takes, in the order a person is asked for them.
	Arguments []SharedCommandArgument
}

// SharedCommandArgument is one argument of a shared command.
//
//numen:api
type SharedCommandArgument struct {
	// Name is the name the argument is passed under in Request.Arguments.
	Name string
	// Title is what a person is asked for.
	Title string
	// Type is string, number or boolean.
	Type string
	// IsRequired is set for an argument every run is given.
	IsRequired bool
}
