package agentrun

import numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

// Program names a command-line agent program and where it may be installed.
type Program struct {
	// Names are the executable names to search for on the PATH and in installation locations.
	Names []string
	// Places are additional glob patterns to search beyond the standard locations.
	Places []string
}

// StandardPlaces returns the standard per-OS locations searched for the named programs.
func StandardPlaces(names ...string) []string {
	var places []string
	for _, name := range names {
		places = append(places,
			"~/.local/bin/"+name,
			"/opt/homebrew/bin/"+name,
			"/usr/local/bin/"+name,
			"~/.nix-profile/bin/"+name,
			"/run/current-system/sw/bin/"+name,
		)
	}
	return places
}

// Discover returns the path of the program and whether it was found installed. Where not found, it
// returns the first name and false.
func Discover(p Program) (string, bool) {
	var places []string
	for _, name := range p.Names {
		places = append(places, "~/.local/bin/"+name)
	}
	places = append(places, p.Places...)
	for _, name := range p.Names {
		places = append(places,
			"/opt/homebrew/bin/"+name,
			"/usr/local/bin/"+name,
			"~/.nix-profile/bin/"+name,
			"/run/current-system/sw/bin/"+name,
		)
	}
	if found := Locate(p.Names, places); found != "" {
		return found, true
	}
	if len(p.Names) > 0 {
		return p.Names[0], false
	}
	return "", false
}

// ProgramCommand returns the command line configured for the named program, or nil.
func ProgramCommand(commands []*numenv1.AgentProgramCommand, program string) []string {
	for _, one := range commands {
		if one.GetProgram() == program {
			return one.GetCommand()
		}
	}
	return nil
}
