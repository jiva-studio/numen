package completion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/settings"
)

// absentLauncher starts no program.
type absentLauncher struct{}

func (absentLauncher) Launch(context.Context, string, []string) (acp.Process, error) {
	return nil, errors.New("no program starts here")
}

type fakeListings struct{}

func (fakeListings) GetListing(context.Context, agents.Agent) acp.Listing { return acp.Listing{} }

func TestRequestLimitIsTheSettingElseTwoMinutes(t *testing.T) {
	values := new(settings.Settings)
	subject, err := New(Dependencies{
		Settings: values, Launcher: absentLauncher{}, Listings: fakeListings{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("default = %v, want 120s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "45"})
	if got := subject.getRequestLimit(); got != 45*time.Second {
		t.Errorf("set to 45 = %v, want 45s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "soon"})
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("invalid value = %v, want 120s", got)
	}
}

func TestNewRefusesMissingDependencies(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Error("New accepted no dependencies")
	}
}

func TestAProgramIsUnreachableWhenNoLookupIsGiven(t *testing.T) {
	subject, err := New(Dependencies{Settings: new(settings.Settings), Launcher: absentLauncher{}, Listings: fakeListings{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = subject.getCallAgent(connection.Call{Type: "acp", Fields: map[string]string{"program": "agent"}})
	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("getCallAgent = %v, want try the next", err)
	}
}
