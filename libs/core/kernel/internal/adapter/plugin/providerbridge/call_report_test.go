package providerbridge_test

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
)

func reportCall(t *testing.T, service *scriptedService) ([]string, error) {
	t.Helper()
	var reported []string
	source := providerbridge.NewCallReportingSource(providerbridge.NewSource(newActivator(t, service)), func(connectionID string) {
		reported = append(reported, connectionID)
	})
	provider, release, err := source.OpenProvider(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, err = provider.Call(t.Context(), "llm.chat", modelcall.ProviderCall{ConnectionID: "c0000wren"}, nil, func([]byte) error { return nil })
	return reported, err
}

func TestCallReportingSourceReportsTheConnectionOfAnAnsweredCall(t *testing.T) {
	reported, err := reportCall(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{done([]byte("finished"))}})

	if err != nil || !slices.Equal(reported, []string{"c0000wren"}) {
		t.Errorf("reported = %v, %v, want the connection", reported, err)
	}
}

func TestCallReportingSourceDoesNotReportAFailedCall(t *testing.T) {
	reported, err := reportCall(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{
		failure(v1.FailureKind_FAILURE_KIND_TRY_NEXT, v1.FailureReason_FAILURE_REASON_UNAVAILABLE, 0),
	}})

	if err == nil || len(reported) != 0 {
		t.Errorf("reported = %v, %v, want a failure and no report", reported, err)
	}
}
