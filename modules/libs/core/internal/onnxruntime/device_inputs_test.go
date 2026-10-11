package onnxruntime_test

import (
	"testing"
)

func TestEmbedderInputsAreValidForAnyEmbeddingModel(t *testing.T) {
	ids := func(name string) tensorSignature {
		return tensorSignature{
			Name:   name,
			Type:   elementInt64,
			Dims:   []int64{dynamicDim, dynamicDim},
			Params: []string{"batch", "sequence"},
		}
	}
	sig := modelSignature{Inputs: []tensorSignature{ids("input_ids"), ids("attention_mask"), ids("token_type_ids")}}

	for _, w := range getEmbedderWorkloads() {
		t.Run(w.name, func(t *testing.T) {
			built, err := buildInputData(sig, w)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]func(int64) bool{
				"input_ids":      func(v int64) bool { return v == fillTokenID && v > 1 },
				"attention_mask": func(v int64) bool { return v == 1 },
				"token_type_ids": func(v int64) bool { return v == 0 },
			} {
				data, ok := built[name].data.([]int64)
				if !ok || len(data) == 0 {
					t.Fatalf("%s holds %T with %d elements, want a non-empty []int64", name, built[name].data, len(data))
				}
				for i, v := range data {
					if !want(v) {
						t.Fatalf("%s[%d] = %d, which is not valid for this input", name, i, v)
					}
				}
			}
		})
	}
}
