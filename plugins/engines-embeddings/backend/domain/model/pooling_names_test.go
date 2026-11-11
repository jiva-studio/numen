package model

import (
	"testing"
)

func TestProfileNamingTheOldPoolingIsRejected(t *testing.T) {
	var reported error
	table := getShipped(
		t,
	).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "h.json", Data: []byte(`{"id":"owner/h","dims":8,"maxTokens":8,"pooling":"head","onnxFile":"m.onnx"}`)}},
		func(_ string, err error) { reported = err },
	)
	if reported == nil {
		t.Error("a profile with the pooling head was not reported")
	}
	if _, err := table.GetLocal("owner/h"); err == nil {
		t.Error("a profile with the pooling head is in the table")
	}
}

func TestShippedProfileNamingTheOldPoolingIsRejected(t *testing.T) {
	origEmbedded := embeddedProfiles
	t.Cleanup(func() { embeddedProfiles = origEmbedded })
	embeddedProfiles = []byte(`[{"id":"x","dims":8,"maxTokens":8,"pooling":"head","onnxFile":"m.onnx"}]`)
	if _, err := GetShippedProfiles(); err == nil {
		t.Error("GetShippedProfiles accepted the pooling head")
	}
}

func TestProfileValidationRejectsTheOldPooling(t *testing.T) {
	p := Profile{ID: "owner/h", Dimensions: 8, MaxTokens: 8, Pooling: "head", OnnxFile: "m.onnx"}
	if err := p.Validate(); err == nil {
		t.Error("Validate accepted the pooling head")
	}
}

func TestModelIdentityRejectsTheOldPooling(t *testing.T) {
	if err := CheckIdentity(ModelInfo{Dimensions: 1024, MaxTokens: 512, Pooling: "head"}, "repo"); err == nil {
		t.Error("CheckIdentity accepted the pooling head")
	}
}
