package machineport

import "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"

// SpecsReader reports what the machine offers.
type SpecsReader interface {
	ReadSpecs() machine.Specs
}
