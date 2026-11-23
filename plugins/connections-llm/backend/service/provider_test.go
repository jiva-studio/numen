package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
)

func TestTheProviderServesEveryOperationOfTheContract(t *testing.T) {
	provider := newTestService(t).built.provider.roles()
	for _, operation := range []string{
		llm.OperationComplete, llm.OperationChat, llm.OperationEmbed, llm.OperationProfile, llm.OperationModels, llm.OperationReasoning,
		llm.OperationRerank,
	} {
		if provider[operation] == nil {
			t.Errorf("the provider does not serve %s", operation)
		}
	}
	if _, err := newService(clock.System{}, models.Lists{}); err == nil {
		t.Error("newService without a report callback succeeded, want error")
	}
}
