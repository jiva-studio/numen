package model

import "testing"

func TestOverridesThatAreSetReplaceTheLocalModelFields(t *testing.T) {
	local := LocalModel{Name: "orig/repo", File: "orig.onnx", Threads: 4, Revision: "abc", Sums: map[string]string{"a": "b"}}
	Overrides{Repository: "custom/repo", Dir: "/custom/dir", File: "custom.onnx", Runtime: "/custom/libonnx.so", Threads: 1}.ApplyToLocal(&local)
	if local.Name != "custom/repo" || local.Revision != "" || local.Sums != nil ||
		local.Dir != "/custom/dir" || local.File != "custom.onnx" || local.Runtime != "/custom/libonnx.so" || local.Threads != 1 {
		t.Fatalf("local = %+v", local)
	}
}

func TestOverridesThatAreNotSetKeepTheLocalModelFields(t *testing.T) {
	local := LocalModel{Name: "kept/repo", Dir: "/kept", File: "kept.onnx", Runtime: "libkept.so", Threads: 4}
	Overrides{}.ApplyToLocal(&local)
	if local.Name != "kept/repo" || local.Dir != "/kept" || local.File != "kept.onnx" || local.Runtime != "libkept.so" || local.Threads != 4 {
		t.Fatalf("local = %+v", local)
	}
}

func TestOverridesReplaceTheModelInfoFields(t *testing.T) {
	info := ModelInfo{Dimensions: 384, MaxTokens: 512, Pooling: "mean"}
	Overrides{Dimensions: 1, MaxTokens: 1, Pooling: "last"}.ApplyToInfo(&info)
	if info.Dimensions != 1 || info.MaxTokens != 1 || info.Pooling != "last" {
		t.Fatalf("info = %+v", info)
	}
	Overrides{}.ApplyToInfo(&info)
	if info.Dimensions != 1 || info.MaxTokens != 1 || info.Pooling != "last" {
		t.Fatalf("empty overrides changed info = %+v", info)
	}
}
