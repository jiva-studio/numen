package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
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
}
