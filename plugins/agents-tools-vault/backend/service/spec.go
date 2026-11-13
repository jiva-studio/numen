package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func createReadSpec(name, title, description string, instructions ...string) tools.Spec {
	return tools.Spec{
		Name: name, Title: title, Description: description, Effect: tools.ReadOnly,
		Instructions: instructions, Requires: nil,
	}
}

func createWriteSpec(name, title, description string, instructions ...string) tools.Spec {
	return tools.Spec{
		Name: name, Title: title, Description: description, Effect: tools.ReadWrite,
		Instructions: instructions, Requires: nil,
	}
}

func requirePermission(spec tools.Spec, permission string) tools.Spec {
	spec.Requires = []string{permission}
	return spec
}

func requireManagePermission(spec tools.Spec) tools.Spec {
	return requirePermission(spec, host.PermissionVaultManage)
}
