package engine

import "testing"

func TestOverridesThatAreSetReplaceTheLocalModelFields(t *testing.T) {
	local := LocalModel{Threads: 4}
	Overrides{Dir: "/custom/dir", Runtime: "/custom/libonnx.so", Threads: 6}.ApplyToLocal(&local)
	if local.Dir != "/custom/dir" || local.Runtime != "/custom/libonnx.so" || local.Threads != 6 {
		t.Fatalf("local = %+v", local)
	}
}

func TestOverridesThatAreNotSetKeepTheLocalModelFields(t *testing.T) {
	local := LocalModel{Dir: "/kept", Runtime: "libkept.so", Threads: 4}
	Overrides{}.ApplyToLocal(&local)
	if local.Dir != "/kept" || local.Runtime != "libkept.so" || local.Threads != 4 {
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
