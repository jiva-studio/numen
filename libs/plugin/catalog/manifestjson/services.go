package manifestjson

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// protocolPackage is the package of the services the core declares.
const protocolPackage = protoreflect.FullName("numen.v1")

func listServiceNameIssues(m manifest.Manifest) []error {
	shared := listProtocolServices()
	var issues []error
	for _, name := range m.Services {
		isOwn := name == m.ID || strings.HasPrefix(name, m.ID+".")
		if !isOwn && !shared[name] {
			issues = append(issues, fmt.Errorf("%w: %q starts with neither the plugin id %q nor the name of a service of the protocol",
				manifest.ErrInvalidService, name, m.ID))
		}
	}
	return issues
}

func listProtocolServices() map[string]bool {
	found := map[string]bool{}
	protoregistry.GlobalFiles.RangeFilesByPackage(protocolPackage, func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			found[string(services.Get(i).FullName())] = true
		}
		return true
	})
	return found
}
