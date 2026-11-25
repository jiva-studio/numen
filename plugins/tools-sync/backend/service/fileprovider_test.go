package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/fileprovider"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
)

func TestFileProvider_AnswersAFailureOfTheMethodWithItsCodeAndPlainText(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	mux := http.NewServeMux()
	mux.Handle(fileprovider.NewHandler(newFileProvider(r)))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)

	for _, method := range []string{"FetchFile", "DropDownload"} {
		request, err := http.NewRequestWithContext(
			t.Context(), http.MethodPost, web.URL+"/numen.v1.FileProviderService/"+method, strings.NewReader(`{"vaultId":"v-1","path":"a.pdf"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		answer, err := web.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		var failure struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		err = json.NewDecoder(answer.Body).Decode(&failure)
		if closeErr := answer.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err != nil {
			t.Fatal(err)
		}

		if want := string(errorwire.GetCode(errNotConfigured)); failure.Code != want {
			t.Errorf("%s failed with code %q, want %q", method, failure.Code, want)
		}
		if want := errorwire.GetText(errNotConfigured); failure.Message != want {
			t.Errorf("%s failed with %q, want %q", method, failure.Message, want)
		}
	}
}
