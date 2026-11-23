package api_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		api.MethodListVaults: {Request: func() any { return &api.ListVaultsRequest{} }, Response: func() any { return &api.ListVaultsResponse{} }},
		api.MethodListRemoteVaults: {
			Request: func() any { return &api.ListRemoteVaultsRequest{} }, Response: func() any { return &api.ListRemoteVaultsResponse{} },
		},
		api.MethodConnectVault: {Request: func() any { return &api.ConnectVaultRequest{} }, Response: func() any { return &api.ConnectVaultResponse{} }},
		api.MethodGetStatus:    {Request: func() any { return &api.GetStatusRequest{} }, Response: func() any { return &api.GetStatusResponse{} }},
		api.MethodGetSubscription: {
			Request: func() any { return &api.GetSubscriptionRequest{} }, Response: func() any { return &api.GetSubscriptionResponse{} },
		},
		api.MethodListPlans:  {Request: func() any { return &api.ListPlansRequest{} }, Response: func() any { return &api.ListPlansResponse{} }},
		api.MethodChangePlan: {Request: func() any { return &api.ChangePlanRequest{} }, Response: func() any { return &api.ChangePlanResponse{} }},
		api.MethodCreateCheckout: {
			Request: func() any { return &api.CreateCheckoutRequest{} }, Response: func() any { return &api.CreateCheckoutResponse{} },
		},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}

func TestEveryMethodHasItsFixturePair(t *testing.T) {
	plugintest.CheckFixturePairs(t, fixtureDir, getContracts())
}

func TestTheServiceAndMethodNames(t *testing.T) {
	if api.Service != "md.numen.tools.sync" || api.MethodListVaults != "ListVaults" || api.MethodListRemoteVaults != "ListRemoteVaults" ||
		api.MethodConnectVault != "ConnectVault" || api.MethodGetStatus != "GetStatus" {
		t.Errorf("service %q with %q, %q, %q, %q", api.Service, api.MethodListVaults, api.MethodListRemoteVaults, api.MethodConnectVault, api.MethodGetStatus)
	}
}

func TestTheValuesOfStateAndMode(t *testing.T) {
	if api.StateNotConnected != "notConnected" || api.StateConnected != "connected" || api.StateCopy != "copy" ||
		api.ModeNew != "new" || api.ModeExisting != "existing" {
		t.Errorf("states %q, %q, %q and modes %q, %q", api.StateNotConnected, api.StateConnected, api.StateCopy, api.ModeNew, api.ModeExisting)
	}
}

func TestTheNamesOfTheSubscriptionMethodsAndPhases(t *testing.T) {
	if api.MethodGetSubscription != "GetSubscription" || api.MethodCreateCheckout != "CreateCheckout" || api.MethodListPlans != "ListPlans" || api.MethodChangePlan != "ChangePlan" {
		t.Errorf("methods %q and %q", api.MethodGetSubscription, api.MethodCreateCheckout)
	}
	if api.PhaseNone != "none" || api.PhaseActive != "active" || api.PhaseGrace != "grace" || api.PhaseReadOnly != "readOnly" || api.PhaseDeleted != "deleted" {
		t.Errorf("phases %q, %q, %q, %q, %q", api.PhaseNone, api.PhaseActive, api.PhaseGrace, api.PhaseReadOnly, api.PhaseDeleted)
	}
}
