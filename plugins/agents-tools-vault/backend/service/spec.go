package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func reading(step tools.Step, name, title, description string, instructions ...string) tools.Spec {
	return tools.Spec{
		Name: name, Title: title, Description: description, Effect: tools.ReadOnly, Step: step,
		Servers: []tools.Server{tools.Full, tools.Reading}, Instructions: instructions, Requires: nil,
	}
}

func writing(step tools.Step, name, title, description string, instructions ...string) tools.Spec {
	return tools.Spec{
		Name: name, Title: title, Description: description, Effect: tools.ReadWrite, Step: step,
		Servers: []tools.Server{tools.Full}, Instructions: instructions, Requires: nil,
	}
}

func requiring(spec tools.Spec, permission string) tools.Spec {
	spec.Requires = []string{permission}
	return spec
}

func managing(spec tools.Spec) tools.Spec {
	return requiring(spec, host.PermissionVaultManage)
}
