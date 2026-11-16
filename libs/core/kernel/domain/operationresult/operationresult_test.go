package operationresult_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/operationresult"
)

func TestTheInputHashDiffersByInstructionAndIsEqualForEqualInputs(t *testing.T) {
	one := operationresult.GetRerankInputHash("tide", "", "the tide comes in")
	if one != operationresult.GetRerankInputHash("tide", "", "the tide comes in") {
		t.Error("equal inputs hash apart")
	}
	if one == operationresult.GetRerankInputHash("tide", "find the answer", "the tide comes in") {
		t.Error("another instruction hashes the same")
	}
	if operationresult.GetRerankInputHash("a\x00b", "", "c") == operationresult.GetRerankInputHash("a", "b", "c") {
		t.Error("the parts are not kept apart")
	}
}

func TestTheRecipeHashIsEqualForTheSameServiceAndDiffersByEachPart(t *testing.T) {
	base := operationresult.RerankRecipe{
		Fingerprint: "md.example/server https://one.test", ProviderVersion: "1.0.0",
		Options: map[string]string{"model": "rerank-1", "b": "x"}, MaxRerankedBytes: 4096,
	}
	same := base
	same.Options = map[string]string{"b": "x", "model": "rerank-1"}
	if operationresult.GetRerankRecipeHash(base) != operationresult.GetRerankRecipeHash(same) {
		t.Error("the same recipe hashes apart")
	}
	changes := []func(r *operationresult.RerankRecipe){
		func(r *operationresult.RerankRecipe) { r.Fingerprint = "md.example/server https://two.test" },
		func(r *operationresult.RerankRecipe) { r.ProviderVersion = "1.0.1" },
		func(r *operationresult.RerankRecipe) { r.Options = map[string]string{"model": "rerank-2", "b": "x"} },
		func(r *operationresult.RerankRecipe) { r.MaxRerankedBytes = 2048 },
	}
	for i, change := range changes {
		other := base
		change(&other)
		if operationresult.GetRerankRecipeHash(base) == operationresult.GetRerankRecipeHash(other) {
			t.Errorf("change %d hashes the same", i)
		}
	}
}

func TestAScoreReadsBackAndAWrongOutputIsRefused(t *testing.T) {
	got, err := operationresult.DecodeScore(operationresult.EncodeScore(-0.25))
	if err != nil || got != -0.25 {
		t.Errorf("DecodeScore = %v, %v", got, err)
	}
	if _, err := operationresult.DecodeScore([]byte{1, 2}); err == nil {
		t.Error("a short output was read as a score")
	}
}
