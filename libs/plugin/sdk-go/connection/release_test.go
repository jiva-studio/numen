package connection

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
)

const operationChat = "llm.chat"

func newReleasingClient(t *testing.T, provider Provider) (numenv1connect.ProviderServiceClient, *processtest.Host) {
	t.Helper()
	host := processtest.NewHost(t)
	mux := http.NewServeMux()
	mux.Handle(NewHandler(provider, host.NewClient(t)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewProviderServiceClient(server.Client(), server.URL), host
}

func detectOneConnection(context.Context, Call) ([]DetectedConnection, error) {
	return []DetectedConnection{{Name: "Atlas", Fields: map[string]string{"model": "atlas-1"}}}, nil
}

func listOneOption(context.Context, Call, OptionsQuery) (ListedOptions, error) {
	return ListedOptions{Options: []Option{{Value: "atlas-1", Title: "Atlas 1"}}}, nil
}

func answerAtOnce(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
	return []byte("answer"), nil
}

func callOperation(t *testing.T, client numenv1connect.ProviderServiceClient, operation string, payload []byte) served {
	t.Helper()
	request := newCallRequest(operation)
	request.Msg.Payload = payload
	got, err := readCallResponse(t, client, request)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestADetectionAnswerReleasesTheProcess(t *testing.T) {
	client, host := newReleasingClient(t, Provider{OperationDetected: ServeDetected(detectOneConnection)})

	if got := callOperation(t, client, OperationDetected, nil); !got.isDone {
		t.Fatalf("call = %+v, want done", got)
	}

	if releases := host.CountReleases(); releases != 1 {
		t.Errorf("releases = %d, want 1", releases)
	}
}

func TestAnOptionListReleasesTheProcess(t *testing.T) {
	client, host := newReleasingClient(t, Provider{"llm.models": ServeOptions(listOneOption)})
	payload, err := proto.Marshal(&v1.OptionsRequest{Operations: []string{operationChat}})
	if err != nil {
		t.Fatal(err)
	}

	if got := callOperation(t, client, "llm.models", payload); !got.isDone {
		t.Fatalf("call = %+v, want done", got)
	}

	if releases := host.CountReleases(); releases != 1 {
		t.Errorf("releases = %d, want 1", releases)
	}
}

func TestAnOperationThatIsNoLookupKeepsTheProcess(t *testing.T) {
	client, host := newReleasingClient(t, Provider{operationChat: answerAtOnce})

	callOperation(t, client, operationChat, nil)

	if releases := host.CountReleases(); releases != 0 {
		t.Errorf("releases = %d, want none", releases)
	}
}

func TestAFailedDetectionKeepsTheProcess(t *testing.T) {
	client, host := newReleasingClient(t, Provider{OperationDetected: ServeDetected(func(context.Context, Call) ([]DetectedConnection, error) {
		return nil, NewStopError(ReasonUnavailable)
	})})

	callOperation(t, client, OperationDetected, nil)

	if releases := host.CountReleases(); releases != 0 {
		t.Errorf("releases = %d, want none", releases)
	}
}

func TestALookupDuringAnOpenCallKeepsTheProcess(t *testing.T) {
	isStarted, finish := make(chan struct{}), make(chan struct{})
	client, host := newReleasingClient(t, Provider{
		OperationDetected: ServeDetected(detectOneConnection),
		operationChat: func(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
			close(isStarted)
			<-finish
			return []byte("answer"), nil
		},
	})
	chatDone := make(chan error)
	go func() {
		_, err := readCallResponse(t, client, newCallRequest(operationChat))
		chatDone <- err
	}()
	<-isStarted

	callOperation(t, client, OperationDetected, nil)
	close(finish)
	if err := <-chatDone; err != nil {
		t.Fatal(err)
	}

	if releases := host.CountReleases(); releases != 0 {
		t.Errorf("releases = %d, want none while a chat was open", releases)
	}
}
