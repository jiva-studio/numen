package hostendpoint

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func getMountedServices() []string {
	return []string{
		numenv1connect.HostDependencyServiceName, numenv1connect.ConnectionCallServiceName,
		numenv1connect.HostAccountServiceName, numenv1connect.HostStoreServiceName,
		numenv1connect.HostTextServiceName, numenv1connect.HostQueueServiceName,
		numenv1connect.HostSearchServiceName, numenv1connect.PluginToolServiceName,
		numenv1connect.HostWindowServiceName, numenv1connect.PluginHealthServiceName,
		numenv1connect.HostProcessServiceName, numenv1connect.LogServiceName,
		numenv1connect.HostVaultServiceName, numenv1connect.VaultsServiceName,
		numenv1connect.HostFileServiceName, numenv1connect.HostNoteServiceName,
		numenv1connect.HostLinkServiceName, numenv1connect.HostCacheServiceName,
	}
}

func getUnservedProcedures() []string {
	return []string{
		numenv1connect.LogServiceGetLogLevelProcedure, numenv1connect.LogServiceSetLogLevelProcedure,
		numenv1connect.LogServiceWatchLogsProcedure, numenv1connect.PluginHealthServiceWatchHealthProcedure,
	}
}

func getMethod(t *testing.T, procedure string) protoreflect.MethodDescriptor {
	t.Helper()
	service, method, _ := strings.Cut(strings.TrimPrefix(procedure, "/"), "/")
	found, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service + "." + method))
	if err != nil {
		t.Fatalf("%s: %v", procedure, err)
	}
	return found.(protoreflect.MethodDescriptor)
}

func TestEveryProcedureOfTheMountedServicesIsListedOrNotServed(t *testing.T) {
	unserved := getUnservedProcedures()
	for _, name := range getMountedServices() {
		found, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		if err != nil {
			t.Fatal(err)
		}
		methods := found.(protoreflect.ServiceDescriptor).Methods()
		for i := range methods.Len() {
			procedure := "/" + name + "/" + string(methods.Get(i).Name())
			_, isListed := procedurePermissions[procedure]
			isUnserved := slices.Contains(unserved, procedure)
			if isListed == isUnserved {
				t.Errorf("%s: listed %v, not served %v, want exactly one", procedure, isListed, isUnserved)
			}
		}
	}
}

func TestEveryListedProcedureNeedsAPermissionOfTheManifestOrNone(t *testing.T) {
	known := map[string]bool{noPermission: true}
	for _, permission := range manifest.ListPluginPermissions() {
		known[permission] = true
	}
	for procedure, permission := range procedurePermissions {
		getMethod(t, procedure)
		if !known[permission] {
			t.Errorf("%s needs %q, which no manifest declares", procedure, permission)
		}
	}
}

func postProcedure(t *testing.T, base, procedure, token string) (int, string) {
	t.Helper()
	method := getMethod(t, procedure)
	contentType, body := "application/json", []byte("{}")
	if method.IsStreamingServer() || method.IsStreamingClient() {
		contentType, body = "application/connect+json", []byte{0, 0, 0, 0, 2, '{', '}'}
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, base+procedure, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Connect-Protocol-Version", "1")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	answer, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(answer)
}

func openEndpoint(t *testing.T, isConfined bool) string {
	t.Helper()
	fake := newCallers()
	fake.confined = isConfined
	endpoint, err := Open(t.Context(), Address, fake.asCallers(), AccountCalls{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	return "http://127.0.0.1:" + strconv.Itoa(endpoint.GetPort())
}

func TestAPluginWithoutThePermissionOfAProcedureIsRefusedBeforeTheHandlerRuns(t *testing.T) {
	base := openEndpoint(t, true)
	for procedure, permission := range procedurePermissions {
		if permission == noPermission {
			continue
		}
		_, answer := postProcedure(t, base, procedure, "token-a")
		if !strings.Contains(answer, "permission_denied") {
			t.Errorf("%s without %s answered %s", procedure, permission, answer)
		}
	}
}

func TestAProcedureNoPermissionListsIsRefusedToAPluginThatHoldsEveryPermission(t *testing.T) {
	base := openEndpoint(t, false)
	for _, procedure := range getUnservedProcedures() {
		if _, answer := postProcedure(t, base, procedure, "token-a"); !strings.Contains(answer, "permission_denied") {
			t.Errorf("%s answered %s, want permission denied", procedure, answer)
		}
	}
}

func TestAnUnknownTokenIsUnauthenticatedBeforeTheTableIsRead(t *testing.T) {
	fake := newCallers()
	gate := permissionGate{endpoint: &Endpoint{callers: fake.asCallers()}}
	header := http.Header{"Authorization": {"Bearer nobody"}}

	err := gate.check("/unlisted.Service/Call", header)

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}
