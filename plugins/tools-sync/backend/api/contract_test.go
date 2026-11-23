package api_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		api.MethodListVaults: {CreateRequest: func() any { return &api.ListVaultsRequest{} }, CreateResponse: func() any { return &api.ListVaultsResponse{} }},
		api.MethodListRemoteVaults: {
			CreateRequest: func() any { return &api.ListRemoteVaultsRequest{} }, CreateResponse: func() any { return &api.ListRemoteVaultsResponse{} },
		},
		api.MethodConnectVault: {CreateRequest: func() any { return &api.ConnectVaultRequest{} }, CreateResponse: func() any { return &api.ConnectVaultResponse{} }},
		api.MethodPreviewConnect: {
			CreateRequest: func() any { return &api.PreviewConnectRequest{} }, CreateResponse: func() any { return &api.PreviewConnectResponse{} },
		},
		api.MethodResolveDeletions: {
			CreateRequest: func() any { return &api.ResolveDeletionsRequest{} }, CreateResponse: func() any { return &api.ResolveDeletionsResponse{} },
		},
		api.MethodGetStatus: {CreateRequest: func() any { return &api.GetStatusRequest{} }, CreateResponse: func() any { return &api.GetStatusResponse{} }},
		api.MethodGetSubscription: {
			CreateRequest: func() any { return &api.GetSubscriptionRequest{} }, CreateResponse: func() any { return &api.GetSubscriptionResponse{} },
		},
		api.MethodListPlans:  {CreateRequest: func() any { return &api.ListPlansRequest{} }, CreateResponse: func() any { return &api.ListPlansResponse{} }},
		api.MethodChangePlan: {CreateRequest: func() any { return &api.ChangePlanRequest{} }, CreateResponse: func() any { return &api.ChangePlanResponse{} }},
		api.MethodCreatePortalSession: {
			CreateRequest: func() any { return &api.CreatePortalSessionRequest{} }, CreateResponse: func() any { return &api.CreatePortalSessionResponse{} },
		},
		api.MethodCreateCheckout: {
			CreateRequest: func() any { return &api.CreateCheckoutRequest{} }, CreateResponse: func() any { return &api.CreateCheckoutResponse{} },
		},
		api.MethodGetReaderCode: {
			CreateRequest: func() any { return &api.GetReaderCodeRequest{} }, CreateResponse: func() any { return &api.GetReaderCodeResponse{} },
		},
		api.MethodReadReaderCode: {
			CreateRequest: func() any { return &api.ReadReaderCodeRequest{} }, CreateResponse: func() any { return &api.ReadReaderCodeResponse{} },
		},
		api.MethodGetShare: {CreateRequest: func() any { return &api.GetShareRequest{} }, CreateResponse: func() any { return &api.GetShareResponse{} }},
		api.MethodAddReader: {
			CreateRequest: func() any { return &api.AddReaderRequest{} }, CreateResponse: func() any { return &api.AddReaderResponse{} },
		},
		api.MethodRemoveReader: {
			CreateRequest: func() any { return &api.RemoveReaderRequest{} }, CreateResponse: func() any { return &api.RemoveReaderResponse{} },
		},
		api.MethodStopShare: {CreateRequest: func() any { return &api.StopShareRequest{} }, CreateResponse: func() any { return &api.StopShareResponse{} }},
		api.MethodReadInvitation: {
			CreateRequest: func() any { return &api.ReadInvitationRequest{} }, CreateResponse: func() any { return &api.ReadInvitationResponse{} },
		},
		api.MethodAcceptInvitation: {
			CreateRequest: func() any { return &api.AcceptInvitationRequest{} }, CreateResponse: func() any { return &api.AcceptInvitationResponse{} },
		},
		api.MethodGetAccountState: {
			CreateRequest: func() any { return &api.GetAccountStateRequest{} }, CreateResponse: func() any { return &api.GetAccountStateResponse{} },
		},
		api.MethodDeleteCloudData: {
			CreateRequest: func() any { return &api.DeleteCloudDataRequest{} }, CreateResponse: func() any { return &api.DeleteCloudDataResponse{} },
		},
		api.MethodRestoreAccount: {
			CreateRequest: func() any { return &api.RestoreAccountRequest{} }, CreateResponse: func() any { return &api.RestoreAccountResponse{} },
		},
		api.MethodExportAccountData: {
			CreateRequest: func() any { return &api.ExportAccountDataRequest{} }, CreateResponse: func() any { return &api.ExportAccountDataResponse{} },
		},
		api.MethodListReceived: {
			CreateRequest: func() any { return &api.ListReceivedRequest{} }, CreateResponse: func() any { return &api.ListReceivedResponse{} },
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
		api.MethodPreviewConnect != "PreviewConnect" || api.MethodConnectVault != "ConnectVault" || api.MethodGetStatus != "GetStatus" {
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

func TestTheNameOfTheMethodThatAnswersPausedRemovalsAndTheirDirections(t *testing.T) {
	if api.MethodResolveDeletions != "ResolveDeletions" {
		t.Errorf("method %q, want ResolveDeletions", api.MethodResolveDeletions)
	}
}

func TestTheNamesOfTheSharingMethodsAndTheirStates(t *testing.T) {
	methods := map[string]string{
		api.MethodGetReaderCode: "GetReaderCode", api.MethodReadReaderCode: "ReadReaderCode", api.MethodGetShare: "GetShare",
		api.MethodAddReader: "AddReader", api.MethodRemoveReader: "RemoveReader", api.MethodStopShare: "StopShare",
		api.MethodReadInvitation: "ReadInvitation", api.MethodAcceptInvitation: "AcceptInvitation", api.MethodListReceived: "ListReceived",
	}
	for got, want := range methods {
		if got != want {
			t.Errorf("method %q, want %q", got, want)
		}
	}
	if api.StateReceived != "received" || api.ShareNotShared != "notShared" || api.ShareCreating != "creating" || api.ShareActive != "active" ||
		api.ReaderActive != "active" || api.ReaderRemoved != "removed" ||
		api.ReceivedReceiving != "receiving" || api.ReceivedReady != "ready" || api.ReceivedEnded != "ended" {
		t.Error("a state of sharing has another word than the frontend reads")
	}
}
